package sluice

import (
	"errors"
	"runtime"
	"sync/atomic"

	"github.com/safing/portmaster/base/log"
	"github.com/safing/portmaster/service/mgr"
	"github.com/safing/portmaster/service/netenv"
	"github.com/safing/portmaster/spn/conf"
)

type SluiceModule struct {
	mgr      *mgr.Manager
	instance instance
}

func (s *SluiceModule) Manager() *mgr.Manager {
	return s.mgr
}

func (s *SluiceModule) Start() error {
	return start()
}

func (s *SluiceModule) Stop() error {
	return stop()
}

var (
	entrypointInfoMsg = []byte("You have reached the local SPN entry port, but your connection could not be matched to an SPN tunnel.\n")

	// EnableListener indicates if it should start the sluice listeners. Must be set at startup.
	EnableListener bool = true
)

func start() error {
	// Windows needs the listeners on all interfaces. On Linux, connections reach the sluice only
	// through the DNAT to 127.0.0.17:717 and [::1]:717 (see nfqueue_linux.go), so it listens there
	// alone: the SPN entry port is not open to the local network.
	ip4, ip6 := "0.0.0.0", "::"
	if runtime.GOOS == "linux" {
		ip4, ip6 = "127.0.0.17", "::1"
	}

	if conf.Integrated() && EnableListener {
		StartSluice("tcp4", ip4+":717")
		StartSluice("udp4", ip4+":717")

		if netenv.IPv6Enabled() {
			StartSluice("tcp6", "["+ip6+"]:717")
			StartSluice("udp6", "["+ip6+"]:717")
		} else {
			log.Warningf("spn/sluice: no IPv6 stack detected, disabling IPv6 SPN entry endpoints")
		}
	}

	return nil
}

func stop() error {
	stopAllSluices()
	return nil
}

var (
	module     *SluiceModule
	shimLoaded atomic.Bool
)

// New returns a new Config module.
func New(instance instance) (*SluiceModule, error) {
	if !shimLoaded.CompareAndSwap(false, true) {
		return nil, errors.New("only one instance allowed")
	}
	m := mgr.New("SluiceModule")
	module = &SluiceModule{
		mgr:      m,
		instance: instance,
	}
	return module, nil
}

type instance interface{}

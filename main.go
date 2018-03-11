package main

import (
	"github.com/codemodus/freenet/coms"
	"github.com/codemodus/sigmon"
	"github.com/sirupsen/logrus"
)

func main() {
	// Wire into and ignore common system signals.
	sm := sigmon.New(nil)
	sm.Run()

	// coms provides info/error logging, done signaling, and a WaitGroup.
	c := coms.New(logrus.New())

	// setup iptables
	closeIptables, err := setupIptables()
	if err != nil {
		c.Error(err)
		return
	}
	defer closeIptables()

	// Conc wraps a lambda in a go routine and handles WaitGroup accounting.
	c.Conc(func() {
		listenTCP(c, 80, false)
	})
	c.Conc(func() {
		listenTCP(c, 443, true)
	})

	// Setup system signal behavior (die on all sigs).
	sm.Set(func(s *sigmon.SignalMonitor) {
		c.Info("goodbye")
		c.Close()
	})

	// Wait for WaitGroup resolution.
	c.Wait()
	// Remove system signal wiring.
	sm.Stop()
}

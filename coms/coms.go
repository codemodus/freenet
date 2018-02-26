package coms

import (
	"sync"
)

// Coms holds types for logging and synchronization.
type Coms struct {
	Logger
	d chan struct{}
	w *sync.WaitGroup
}

// New returns a fully prepared *Coms. If nil is provided as the Logger, a
// *voidLog will be used.
func New(log Logger) *Coms {
	if log == nil {
		log = &voidLog{}
	}

	return &Coms{
		Logger: log,
		d:      make(chan struct{}),
		w:      &sync.WaitGroup{},
	}
}

// Done returns the channel used for signaling that the application should be
// considered "done" and begin cleanup.
func (c *Coms) Done() <-chan struct{} {
	return c.d
}

// Conc can be used in place of a simple "go" statement to ease synchronization.
func (c *Coms) Conc(fn func()) {
	c.w.Add(1)

	go func() {
		fn()

		c.w.Done()
	}()
}

// Close closes the *Coms.done channel.
func (c *Coms) Close() {
	close(c.d)
}

// Wait blocks until the WaitGroup resolves.
func (c *Coms) Wait() {
	c.w.Wait()
}

// Logger describes basic logging functions.
type Logger interface {
	Info(...interface{})
	Infof(string, ...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
}

// voidLog fulfills the Logger interface through the discarding of arguments.
type voidLog struct{}

func (l *voidLog) Info(...interface{})           {}
func (l *voidLog) Infof(string, ...interface{})  {}
func (l *voidLog) Error(...interface{})          {}
func (l *voidLog) Errorf(string, ...interface{}) {}

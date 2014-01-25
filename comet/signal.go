package main

import (
	"os"
	"os/signal"
	"syscall"
)

// InitSignal register signals handler.
func InitSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT)

	// Block until a signal is received.
	// TODO
	for {
		s := <-c
		switch s {
		case syscall.SIGHUP:
		// reload
		case syscall.SIGQUIT:
			// quit
			return
		default:
			Log.Debug("get a signal %d", s)
		}
	}
}

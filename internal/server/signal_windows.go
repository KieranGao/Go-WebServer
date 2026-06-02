//go:build windows

package server

import (
	"os"
	"os/signal"
)

// waitForShutdownSignal blocks until SIGINT (Ctrl+C) is received.
func waitForShutdownSignal() os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	return <-quit
}

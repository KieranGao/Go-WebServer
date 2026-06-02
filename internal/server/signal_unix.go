//go:build !windows

package server

import (
	"os"
	"os/signal"
	"syscall"
)

// waitForShutdownSignal blocks until SIGINT or SIGTERM is received.
func waitForShutdownSignal() os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return <-quit
}

//go:build windows

package server

import (
	"os"
	"os/signal"
)

// waitForShutdownSignal 阻塞直到收到 SIGINT（Ctrl+C）信号
func waitForShutdownSignal() os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	return <-quit
}

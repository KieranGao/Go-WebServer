//go:build !windows

package server

import (
	"os"
	"os/signal"
	"syscall"
)

// waitForShutdownSignal 阻塞直到收到 SIGINT 或 SIGTERM 信号
func waitForShutdownSignal() os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return <-quit
}

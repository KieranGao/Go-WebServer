package main

import (
	"flag"
	"log"
	"log/slog"

	"go-webserver/internal/config"
	xlog "go-webserver/internal/log"
	"go-webserver/internal/server"
)

func main() {
	configPath := flag.String("config", "config.conf", "path to config file")
	port := flag.Int("p", 0, "override port (0 = use config file)")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadFile(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if *port > 0 {
		cfg.Port = *port
	}

	// Initialize logger
	var shutdownLogger func()
	if cfg.OpenLog {
		logger, shutdown, err := xlog.New(
			cfg.LogFile,
			xlog.Level(cfg.LogLevel),
			cfg.LogQueueSize,
			cfg.LogFlushInterval,
		)
		if err != nil {
			log.Fatalf("init logger: %v", err)
		}
		slog.SetDefault(logger)
		shutdownLogger = shutdown
	} else {
		slog.SetDefault(xlog.StdoutLogger(xlog.Level(cfg.LogLevel)))
		shutdownLogger = func() {}
	}
	defer shutdownLogger()

	// Create and run server
	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		log.Fatal(err)
	}
	defer srv.Close()

	if err := srv.Run(); err != nil {
		slog.Error("server error", "error", err)
		log.Fatal(err)
	}
}

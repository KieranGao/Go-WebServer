package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration, mirroring the C++ Config singleton.
type Config struct {
	// Network
	Port             int
	OptLinger        bool
	TriggerMode      int // 1-4: LT/ET combinations
	ThreadNum        int
	MaxBodySize      int // bytes
	ConnectionTimeout int // seconds

	// Logging
	LogFile          string
	LogLevel         int // 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR
	LogQueueSize     int
	LogFlushInterval int // seconds
	OpenLog          bool

	// Database
	ConnPoolSize   int
	DBHost         string
	DBPort         int
	DBUser         string
	DBPassword     string
	DBName         string

	// Resources
	ResourceRoot string
}

// DefaultConfig returns a Config with sensible defaults matching config.conf.
func DefaultConfig() *Config {
	return &Config{
		Port:              9999,
		OptLinger:         false,
		TriggerMode:       4,
		ThreadNum:         64,
		MaxBodySize:       1 << 20, // 1MB
		ConnectionTimeout: 60,
		LogFile:           "log/webserver",
		LogLevel:          1,
		LogQueueSize:      1024,
		LogFlushInterval:  3,
		OpenLog:           true,
		ConnPoolSize:      16,
		DBHost:            "127.0.0.1",
		DBPort:            3306,
		DBUser:            "root",
		DBPassword:        "password",
		DBName:            "webserver",
		ResourceRoot:      "resources/",
	}
}

// LoadFile parses a config.conf file (key = value format, # comments).
func LoadFile(path string) (*Config, error) {
	cfg := DefaultConfig()

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "port":
			cfg.Port, _ = strconv.Atoi(val)
		case "opt_linger":
			cfg.OptLinger = val == "true"
		case "trigger_mode":
			cfg.TriggerMode, _ = strconv.Atoi(val)
		case "thread_num":
			cfg.ThreadNum, _ = strconv.Atoi(val)
		case "max_body_size":
			cfg.MaxBodySize, _ = strconv.Atoi(val)
		case "connection_timeout":
			cfg.ConnectionTimeout, _ = strconv.Atoi(val)
		case "log_file":
			cfg.LogFile = val
		case "log_level":
			cfg.LogLevel, _ = strconv.Atoi(val)
		case "log_queue_size":
			cfg.LogQueueSize, _ = strconv.Atoi(val)
		case "log_flush_interval":
			cfg.LogFlushInterval, _ = strconv.Atoi(val)
		case "open_log":
			cfg.OpenLog = val == "true"
		case "connection_pool_size":
			cfg.ConnPoolSize, _ = strconv.Atoi(val)
		case "db_host":
			cfg.DBHost = val
		case "db_port":
			cfg.DBPort, _ = strconv.Atoi(val)
		case "db_user":
			cfg.DBUser = val
		case "db_password":
			cfg.DBPassword = val
		case "db_name":
			cfg.DBName = val
		case "resource_root":
			cfg.ResourceRoot = val
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	return cfg, nil
}

package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// 网络配置
	Port              int // 服务监听端口
	ConnectionTimeout int // 连接超时时间，单位：秒

	// 日志配置
	LogFile          string // 日志文件路径
	LogLevel         int    // 日志级别：0=DEBUG, 1=INFO, 2=WARN, 3=ERROR
	LogQueueSize     int    // 日志队列大小
	LogFlushInterval int    // 日志刷新间隔，单位：秒
	OpenLog          bool   // 是否开启日志

	// 数据库配置
	ConnPoolSize int    // 数据库连接池大小
	DBHost       string // 数据库地址
	DBPort       int    // 数据库端口
	DBUser       string // 数据库用户名
	DBPassword   string // 数据库密码
	DBName       string // 数据库名称

	// 资源文件
	ResourceRoot string // 静态资源根目录
}

// 默认配置
func DefaultConfig() *Config {
	return &Config{
		Port:              9999,
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

	// 逐行读取
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
		// 包装error使用%w可以通过上游解包
		// 自定义的error用%s即可
		return nil, fmt.Errorf("read config file: %w", err)
	}
	return cfg, nil
}

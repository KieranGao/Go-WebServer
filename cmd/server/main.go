package main

import (
	"flag"

	"log"
	"log/slog"

	"go-webserver/internal/config"   // 导入配置
	xlog "go-webserver/internal/log" // 导入自定义的日志系统，起别名为xlog
	"go-webserver/internal/server"
)

func main() {
	// flag 是 Go 官方自带的命令行参数解析工具
	// 变量 := flag.类型("参数名", 默认值, "帮助说明")
	// 需调用解析 flag.Parse()
	// 定义返回的是 指针，所以要加 * 取值：
	configPath := flag.String("config", "config.conf", "path to config file")
	port := flag.Int("p", 0, "override port (0 = use config file)")
	flag.Parse()

	// 解析配置参数
	cfg, err := config.LoadFile(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if *port > 0 {
		cfg.Port = *port
	}

	// 初始化logger
	var shutdownLogger func()
	if cfg.OpenLog { // 日志开启
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
	} else { // 否则输出到控制台
		slog.SetDefault(xlog.StdoutLogger(xlog.Level(cfg.LogLevel)))
		shutdownLogger = func() {}
	}
	defer shutdownLogger()

	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		log.Fatal(err)
	}
	defer srv.Close()

	err = srv.Run() // 开启服务器
	if err != nil {
		slog.Error("server error", "error", err)
		log.Fatal(err)
	}
}

package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go-webserver/internal/config"
	"go-webserver/internal/db"
	"go-webserver/internal/handler"
	"go-webserver/internal/middleware"
)

// Server 持有所有依赖和 HTTP 服务器实例
type Server struct {
	cfg  *config.Config
	db   *db.DB
	http *http.Server
}

// New 根据配置创建服务器，初始化数据库，注册路由和中间件
func New(cfg *config.Config) (*Server, error) {
	// 初始化数据库
	dbConn, err := db.Open(db.Config{
		Host:         cfg.DBHost,
		Port:         cfg.DBPort,
		User:         cfg.DBUser,
		Password:     cfg.DBPassword,
		DBName:       cfg.DBName,
		MaxOpenConns: cfg.ConnPoolSize,
	})
	if err != nil {
		slog.Warn("database unavailable, running without auth", "error", err)
		dbConn = nil
	}

	s := &Server{
		cfg: cfg,
		db:  dbConn,
	}

	// 注册路由
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// 应用中间件链：recovery → logging → timeout → handler
	var root http.Handler = mux
	root = middleware.Timeout(time.Duration(cfg.ConnectionTimeout) * time.Second)(root)
	root = middleware.Log(root)
	root = middleware.Recovery(root)

	s.http = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      root,
		ReadTimeout:  time.Duration(cfg.ConnectionTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.ConnectionTimeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s, nil
}

// methodRoute 根据请求方法分发：GET 走静态文件处理器，POST 走认证处理器
func methodRoute(getHandler, postHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postHandler(w, r)
			return
		}
		getHandler(w, r)
	}
}

// registerRoutes 注册所有 HTTP 路由
func (s *Server) registerRoutes(mux *http.ServeMux) {
	auth := &handler.Auth{DB: s.db}
	staticH := handler.Static(s.cfg.ResourceRoot)

	mux.HandleFunc("/health", handler.Health())

	// /login 和 /register：GET 返回 HTML 页面，POST 处理认证逻辑
	mux.HandleFunc("/login", methodRoute(staticH, auth.Login()))
	mux.HandleFunc("/register", methodRoute(staticH, auth.Register()))

	// 静态文件 — 兜底路由
	mux.HandleFunc("/", staticH)
}

// Run 启动服务器，阻塞直到收到关闭信号
func (s *Server) Run() error {
	// 用于接收服务器错误的通道
	errCh := make(chan error, 1)

	// 在 goroutine 中启动服务器
	go func() {
		slog.Info("HTTP server listening", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// 等待关闭信号（平台相关）
	sig := waitForShutdownSignal()
	slog.Info("shutdown signal received", "signal", sig)

	// 优雅关闭，带超时
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	slog.Info("server stopped gracefully")
	return <-errCh
}

// Close 释放所有资源
func (s *Server) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RayTheCrack/Go-WebServer/internal/config"
	"github.com/RayTheCrack/Go-WebServer/internal/db"
	"github.com/RayTheCrack/Go-WebServer/internal/handler"
	"github.com/RayTheCrack/Go-WebServer/internal/middleware"
)

// Server holds all dependencies and the HTTP server.
type Server struct {
	cfg  *config.Config
	db   *db.DB
	http *http.Server
}

// New creates a Server from config, initializes DB, wires routes and middleware.
func New(cfg *config.Config) (*Server, error) {
	// Initialize database (optional — server runs without it if connection fails)
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

	// Build routes
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Apply middleware chain: recovery → logging → timeout → handler
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

// methodRoute dispatches GET to the static handler and POST to the auth handler.
func methodRoute(getHandler, postHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postHandler(w, r)
			return
		}
		getHandler(w, r)
	}
}

// registerRoutes sets up all HTTP routes.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	auth := &handler.Auth{DB: s.db}
	staticH := handler.Static(s.cfg.ResourceRoot)

	mux.HandleFunc("/health", handler.Health())

	// /login and /register: GET serves the HTML form, POST handles auth
	mux.HandleFunc("/login", methodRoute(staticH, auth.Login()))
	mux.HandleFunc("/register", methodRoute(staticH, auth.Register()))

	// Static files — catch-all for everything else
	mux.HandleFunc("/", staticH)
}

// Run starts the server and blocks until a shutdown signal is received.
func (s *Server) Run() error {
	// Channel to receive errors from the server
	errCh := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		slog.Info("HTTP server listening", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for interrupt signal (SIGINT/SIGTERM), matching C++ SIGINT handler
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutdown signal received", "signal", sig)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	slog.Info("server stopped gracefully")
	return <-errCh
}

// Close releases all resources.
func (s *Server) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB wraps *sql.DB with domain-specific methods.
type DB struct {
	*sql.DB
}

// Config holds database connection parameters.
type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	MaxOpenConns int
}

// Open creates a new database connection pool.
// Returns nil, nil if maxConns is 0 (matching C++ behavior of running without DB).
func Open(cfg Config) (*DB, error) {
	if cfg.MaxOpenConns == 0 {
		slog.Warn("database disabled (connection pool size = 0)")
		return nil, nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("database connected", "host", cfg.Host, "port", cfg.Port, "db", cfg.DBName, "pool_size", cfg.MaxOpenConns)
	return &DB{sqlDB}, nil
}

// usernameRe validates that usernames contain only alphanumeric chars and underscores.
var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// LoginResult represents the outcome of a login attempt.
type LoginResult int

const (
	LoginSuccess LoginResult = iota
	LoginUserNotFound
	LoginWrongPassword
)

// Login checks username/password against the database.
func (db *DB) Login(username, password string) (LoginResult, error) {
	if db == nil {
		return LoginUserNotFound, fmt.Errorf("database not available")
	}
	if !usernameRe.MatchString(username) {
		return LoginUserNotFound, fmt.Errorf("invalid username format")
	}

	var storedPassword string
	err := db.QueryRow("SELECT password FROM user WHERE username = ?", username).Scan(&storedPassword)
	if err == sql.ErrNoRows {
		return LoginUserNotFound, nil
	}
	if err != nil {
		return LoginUserNotFound, fmt.Errorf("query user: %w", err)
	}

	if storedPassword != password {
		return LoginWrongPassword, nil
	}
	return LoginSuccess, nil
}

// Register inserts a new user. Returns an error if the username already exists.
func (db *DB) Register(username, password string) error {
	if db == nil {
		return fmt.Errorf("database not available")
	}
	if !usernameRe.MatchString(username) {
		return fmt.Errorf("invalid username: must be alphanumeric or underscore")
	}

	_, err := db.Exec("INSERT INTO user (username, password) VALUES (?, ?)", username, password)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

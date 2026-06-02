package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB 包装了 *sql.DB，提供业务相关的数据库操作方法
type DB struct {
	*sql.DB
}

type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	MaxOpenConns int
}

// Open 创建数据库连接池
// 如果 maxConns = 0，返回 nil,
func Open(cfg Config) (*DB, error) {
	if cfg.MaxOpenConns == 0 {
		slog.Warn("database disabled (connection pool size = 0)")
		return nil, nil
	}

	// 构建数据库连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	defer sqlDB.Close()

	// 测试连接是否可用
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("database connected", "host", cfg.Host, "port", cfg.Port, "db", cfg.DBName, "pool_size", cfg.MaxOpenConns)
	return &DB{sqlDB}, nil
}

// usernameRe 用户名正则：只允许字母、数字、下划线
var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// LoginResult 登录结果枚举
type LoginResult int

const (
	LoginSuccess       LoginResult = iota // 0
	LoginUserNotFound                     // 1
	LoginWrongPassword                    // 2
)

// Login 校验用户名密码，返回登录结果
func (db *DB) Login(username, password string) (LoginResult, error) {
	if db == nil {
		return LoginUserNotFound, fmt.Errorf("database not available")
	}
	if !usernameRe.MatchString(username) {
		return LoginUserNotFound, fmt.Errorf("invalid username format")
	}

	var storedPassword string
	// 使用 ？ 占位后传入参数，防止SQL注入 mysql驱动会对参数自动转义、消毒
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

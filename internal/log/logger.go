package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Level 将配置中的 log_level (0-3) 映射为 slog 日志级别
func Level(n int) slog.Leveler {
	switch n {
	case 0:
		return slog.LevelDebug
	case 1:
		return slog.LevelInfo
	case 2:
		return slog.LevelWarn
	case 3:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// asyncHandler 实现 slog.Handler 接口，缓冲日志记录并定期刷盘
// 模仿 C++ BlockDeque + worker 线程的设计
type asyncHandler struct {
	level   slog.Leveler
	file    *os.File
	ch      chan string
	done    chan struct{}
	mu      sync.Mutex
	buf     strings.Builder
	flushMs int64 // 刷盘间隔，单位：毫秒
}

// New 创建异步文件日志记录器
// 返回 slog.Logger 和关闭函数
func New(logFile string, level slog.Leveler, queueSize, flushInterval int) (*slog.Logger, func(), error) {
	// 确保日志目录存在
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return nil, nil, fmt.Errorf("create log dir: %w", err)
	}

	f, err := os.OpenFile(logFile+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	h := &asyncHandler{
		level:   level,
		file:    f,
		ch:      make(chan string, queueSize),
		done:    make(chan struct{}),
		flushMs: int64(flushInterval) * 1000,
	}

	// 工作 goroutine：从 channel 读取，缓冲，定期刷盘
	go h.worker()

	logger := slog.New(h)
	shutdown := func() {
		close(h.ch)
		<-h.done
		f.Close()
	}

	return logger, shutdown, nil
}

// Enabled 实现 slog.Handler 接口
func (h *asyncHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle 实现 slog.Handler 接口，格式化日志记录并发送到 channel
func (h *asyncHandler) Handle(_ context.Context, r slog.Record) error {
	ts := r.Time.Format("2006-01-02 15:04:05.000")
	levelStr := levelToString(r.Level)

	var sb strings.Builder
	sb.WriteString(ts)
	sb.WriteString(" [")
	sb.WriteString(levelStr)
	sb.WriteString("] ")
	sb.WriteString(r.Message)

	r.Attrs(func(a slog.Attr) bool {
		sb.WriteString(" ")
		sb.WriteString(a.Key)
		sb.WriteString("=")
		sb.WriteString(a.Value.String())
		return true
	})
	sb.WriteString("\n")

	// 发送到 channel
	h.ch <- sb.String()
	return nil
}

// WithAttrs 实现 slog.Handler 接口
func (h *asyncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // 简化实现：属性在每条记录中处理
}

// WithGroup 实现 slog.Handler 接口
func (h *asyncHandler) WithGroup(name string) slog.Handler {
	return h
}

// worker 从 channel 读取格式化字符串，缓冲后定期刷盘
func (h *asyncHandler) worker() {
	defer close(h.done)

	ticker := time.NewTicker(time.Duration(h.flushMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-h.ch:
			if !ok {
				// channel 已关闭 — 刷盘剩余内容后退出
				h.flush()
				return
			}
			h.mu.Lock()
			h.buf.WriteString(msg)
			needFlush := h.buf.Len() >= 4096
			h.mu.Unlock()

			if needFlush {
				h.flush()
			}
		case <-ticker.C:
			h.flush()
		}
	}
}

// flush 将缓冲内容写入文件
func (h *asyncHandler) flush() {
	h.mu.Lock()
	if h.buf.Len() == 0 { // buffer中没有日志
		h.mu.Unlock()
		return
	}
	data := h.buf.String()
	h.buf.Reset() // 清空buffer
	h.mu.Unlock()

	h.file.WriteString(data)
	h.file.Sync() // 强制落盘
}

func levelToString(level slog.Level) string {
	switch {
	case level <= slog.LevelDebug:
		return "DEBUG"
	case level <= slog.LevelInfo:
		return "INFO"
	case level <= slog.LevelWarn:
		return "WARN"
	default:
		return "ERROR"
	}
}

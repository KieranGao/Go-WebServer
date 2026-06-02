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

// Level maps config log_level (0-3) to slog levels.
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

// asyncHandler is a slog.Handler that buffers log records and flushes
// them to a file periodically, mimicking the C++ BlockDeque + worker thread design.
type asyncHandler struct {
	level   slog.Leveler
	file    *os.File
	ch      chan string
	done    chan struct{}
	mu      sync.Mutex
	buf     strings.Builder
	flushMs int64 // flush interval in milliseconds
}

// New creates an async file-based logger.
// Returns the slog.Logger and a shutdown function.
func New(logFile string, level slog.Leveler, queueSize, flushInterval int) (*slog.Logger, func(), error) {
	// Ensure log directory exists
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

	// Worker goroutine: reads from channel, buffers, flushes periodically
	go h.worker()

	logger := slog.New(h)
	shutdown := func() {
		close(h.ch)
		<-h.done
		f.Close()
	}

	return logger, shutdown, nil
}

// Enabled implements slog.Handler.
func (h *asyncHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle implements slog.Handler. Formats the record and sends to channel.
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

	// Non-blocking send; drop if queue is full (same as C++ fallback to stdout)
	select {
	case h.ch <- sb.String():
	default:
		fmt.Fprint(os.Stdout, sb.String())
	}
	return nil
}

// WithAttrs implements slog.Handler.
func (h *asyncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // simplified: attrs handled per-record
}

// WithGroup implements slog.Handler.
func (h *asyncHandler) WithGroup(name string) slog.Handler {
	return h
}

// worker reads formatted strings from ch, buffers them, and flushes periodically.
func (h *asyncHandler) worker() {
	defer close(h.done)

	ticker := time.NewTicker(time.Duration(h.flushMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-h.ch:
			if !ok {
				// Channel closed — flush remaining and exit
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

// flush writes buffered content to the file.
func (h *asyncHandler) flush() {
	h.mu.Lock()
	if h.buf.Len() == 0 {
		h.mu.Unlock()
		return
	}
	data := h.buf.String()
	h.buf.Reset()
	h.mu.Unlock()

	h.file.WriteString(data)
	h.file.Sync()
}

// StdoutLogger returns a logger that writes to stdout (for when logging to file is disabled).
func StdoutLogger(level slog.Leveler) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
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

// MultiHandler fans out log records to multiple handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			h.Handle(ctx, r)
		}
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithAttrs(attrs)
	}
	return NewMultiHandler(hs...)
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithGroup(name)
	}
	return NewMultiHandler(hs...)
}

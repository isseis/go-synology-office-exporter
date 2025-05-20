package logger

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Logger is the interface for application-wide logging.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	FlushWebhook() error
}

// hybridLogger outputs to stdout in real-time and buffers logs for webhook.
type hybridLogger struct {
	webhookBuffer []slog.Record
	mu            sync.Mutex
	minLevel      Level
	webhookURL    string
	appName       string
	env           string
}

// NewHybridLogger creates a new hybrid logger.
func NewHybridLogger(cfg Config) Logger {
	return &hybridLogger{
		minLevel:   cfg.Level,
		webhookURL: cfg.WebhookURL,
		appName:    cfg.AppName,
		env:        cfg.Environment,
	}
}

// sendToWebhook is a stub for now to allow build. Implement as needed.
func sendToWebhook(webhookURL, appName, env string, logs []slog.Record) error {
	return fmt.Errorf("sendToWebhook not implemented yet")
}

func (h *hybridLogger) log(level slog.Level, msg string, args ...interface{}) {
	if levelFromSlog(level) < h.minLevel {
		return
	}
	rec := slog.NewRecord(time.Now(), level, msg, 0)
	rec.Add(args...)
	_ = slog.Default().Handler().Handle(context.Background(), rec)
	if h.webhookURL != "" {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.webhookBuffer = append(h.webhookBuffer, rec)
	}
}

func (h *hybridLogger) Debug(msg string, args ...interface{}) { h.log(slog.LevelDebug, msg, args...) }
func (h *hybridLogger) Info(msg string, args ...interface{})  { h.log(slog.LevelInfo, msg, args...) }
func (h *hybridLogger) Warn(msg string, args ...interface{})  { h.log(slog.LevelWarn, msg, args...) }
func (h *hybridLogger) Error(msg string, args ...interface{}) { h.log(slog.LevelError, msg, args...) }
func (h *hybridLogger) FlushWebhook() error {
	h.mu.Lock()
	if h.webhookURL == "" || len(h.webhookBuffer) == 0 {
		h.mu.Unlock()
		return nil
	}
	logs := make([]slog.Record, len(h.webhookBuffer))
	copy(logs, h.webhookBuffer)
	h.webhookBuffer = h.webhookBuffer[:0]
	h.mu.Unlock()
	return sendToWebhook(h.webhookURL, h.appName, h.env, logs)
}

// levelFromSlog converts slog.Level to our Level type
func levelFromSlog(lvl slog.Level) Level {
	switch lvl {
	case slog.LevelDebug:
		return LevelDebug
	case slog.LevelWarn:
		return LevelWarn
	case slog.LevelError:
		return LevelError
	default:
		return LevelInfo
	}
}

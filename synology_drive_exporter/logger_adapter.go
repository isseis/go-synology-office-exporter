package synology_drive_exporter

import "github.com/isseis/go-synology-office-exporter/logger"

// loggerAdapter adapts logger.Logger to synology_drive_exporter.Logger interface.
type loggerAdapter struct {
	logger logger.Logger
}

// NewLoggerAdapter creates a new adapter that wraps logger.Logger.
func NewLoggerAdapter(log logger.Logger) Logger {
	return &loggerAdapter{logger: log}
}

func (a *loggerAdapter) Debug(msg string, args ...any) {
	a.logger.Debug(msg, args...)
}

func (a *loggerAdapter) Info(msg string, args ...any) {
	a.logger.Info(msg, args...)
}

func (a *loggerAdapter) Warn(msg string, args ...any) {
	a.logger.Warn(msg, args...)
}

func (a *loggerAdapter) Error(msg string, args ...any) {
	a.logger.Error(msg, args...)
}

func (a *loggerAdapter) FlushWebhook() error {
	return a.logger.FlushWebhook()
}

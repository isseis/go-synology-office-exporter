package logger

import (
	"flag"
	"io"
)

// Command line flags
var (
	logLevelFlag   = flag.String("log-level", "", "Log level (debug, info, warn, error)")
	webhookURLFlag = flag.String("webhook-url", "", "Webhook URL for logging")
	appNameFlag    = flag.String("app-name", "", "Application name")
	envFlag        = flag.String("env", "", "Environment (development, staging, production)")
)

// RegisterFlags is a no-op function kept for backward compatibility.
// Flags are now registered at package initialization time.
// This function will be removed in a future version.
func RegisterFlags() {}

// Level represents the severity of the log message.
type Level int

const (
	// LevelDebug is for debug-level messages.
	LevelDebug Level = iota
	// LevelInfo is for informational messages.
	LevelInfo
	// LevelWarn is for warning messages.
	LevelWarn
	// LevelError is for error messages.
	LevelError
)

// Config holds configuration for the logger.
type Config struct {
	Level       Level     // Log level
	WebhookURL  string    // Webhook URL for sending logs
	AppName     string    // Application name
	Environment string    // Environment (development, staging, production)
	Output      io.Writer // Output destination for stdout (for testing)
}

// ParseLevel parses a string into a Level (defaults to LevelInfo).
func ParseLevel(lvl string) Level {
	switch lvl {
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

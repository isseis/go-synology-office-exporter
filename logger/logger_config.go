package logger

import "io"

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

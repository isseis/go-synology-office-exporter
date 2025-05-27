package logger

import (
	"os"
	"strings"
)

// LoadConfig loads logger config from flags and environment variables.
// Flags take precedence over environment variables.
// The caller must call flag.Parse() before calling this function.
func LoadConfig() (*Config, error) {
	// Start with default values
	levelStr := ""
	webhookURL := ""
	appName := ""
	envName := ""

	// Use flag values if they were set
	if logLevelFlag != nil && *logLevelFlag != "" {
		levelStr = *logLevelFlag
	}
	if webhookURLFlag != nil && *webhookURLFlag != "" {
		webhookURL = *webhookURLFlag
	}
	if appNameFlag != nil && *appNameFlag != "" {
		appName = *appNameFlag
	}
	if envFlag != nil && *envFlag != "" {
		envName = *envFlag
	}

	// Fall back to environment variables if flags not set
	if levelStr == "" {
		levelStr = getEnv("LOG_LEVEL", "warn")
	}
	if webhookURL == "" {
		webhookURL = os.Getenv("LOG_WEBHOOK_URL")
	}
	if appName == "" {
		appName = getEnv("APP_NAME", "synology-office-exporter")
	}
	if envName == "" {
		envName = getEnv("ENV", "development")
	}

	level := ParseLevel(strings.ToLower(levelStr))

	return &Config{
		Level:       level,
		WebhookURL:  webhookURL,
		AppName:     appName,
		Environment: envName,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// EnvVarHelp represents information about an environment variable
// that can be used for generating help messages.
type EnvVarHelp struct {
	Name        string
	Description string
}

// GetEnvVarsHelp returns a slice of environment variable help information
// that the logger package uses.
func GetEnvVarsHelp() []EnvVarHelp {
	return []EnvVarHelp{
		{"LOG_LEVEL", "Log level (debug, info, warn, error)"},
		{"LOG_WEBHOOK_URL", "Webhook URL for logging"},
		{"APP_NAME", "Application name"},
		{"ENV", "Environment (development, staging, production)"},
	}
}

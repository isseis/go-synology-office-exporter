package logger

import (
	"flag"
	"fmt"
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
		levelStr = getEnv("LOG_LEVEL", "info")
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
		Output:      nil, // Set by caller if needed
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Usage prints the logger config flag usage and environment variable info.
func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintln(flag.CommandLine.Output(), "\nEnvironment Variables:")
	fmt.Fprintln(flag.CommandLine.Output(), "  LOG_LEVEL       Log level (debug, info, warn, error)")
	fmt.Fprintln(flag.CommandLine.Output(), "  LOG_WEBHOOK_URL Webhook URL for logging")
	fmt.Fprintln(flag.CommandLine.Output(), "  APP_NAME        Application name")
	fmt.Fprintln(flag.CommandLine.Output(), "  ENV             Environment (development, staging, production)")
}

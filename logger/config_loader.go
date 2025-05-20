package logger

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// LoadConfig loads logger config from flags and environment variables.
// Flags take precedence over environment variables.
// NOTE: The caller (main.go) must call flag.Parse() before calling LoadConfig.
func LoadConfig() (*Config, error) {
	var (
		levelStr   string
		webhookURL string
		appName    string
		envName    string
	)
	flag.StringVar(&levelStr, "log-level", "", "Log level (debug, info, warn, error)")
	flag.StringVar(&webhookURL, "webhook-url", "", "Webhook URL for logging")
	flag.StringVar(&appName, "app-name", "", "Application name")
	flag.StringVar(&envName, "env", "", "Environment (development, staging, production)")
	// Do NOT call flag.Parse() here; main.go is responsible for parsing flags.

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

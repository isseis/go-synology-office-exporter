//go:build test

package synology_drive_exporter

import (
	"testing"

	"github.com/isseis/go-synology-office-exporter/logger"
)

// TestExporterWithLogLevel tests that the WithLogLevel option works correctly.
func TestExporterWithLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    logger.Level
		expected logger.Level
	}{
		{"Debug level", logger.LevelDebug, logger.LevelDebug},
		{"Info level", logger.LevelInfo, logger.LevelInfo},
		{"Warn level", logger.LevelWarn, logger.LevelWarn},
		{"Error level", logger.LevelError, logger.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := NewExporterWithDependencies(
				nil, "", nil,
				WithLogLevel(tt.level),
			)

			if exporter.logLevel != tt.expected {
				t.Errorf("Expected log level %d, got %d", tt.expected, exporter.logLevel)
			}
		})
	}
}

// TestDefaultLogLevel tests that the default log level is set correctly.
func TestDefaultLogLevel(t *testing.T) {
	exporter := NewExporterWithDependencies(nil, "", nil)

	if exporter.logLevel != logger.LevelWarn {
		t.Errorf("Expected default log level to be %d (LevelWarn), got %d", logger.LevelWarn, exporter.logLevel)
	}
}

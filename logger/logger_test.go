package logger

import (
	"os"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	cases := []struct {
		in   string
		want Level
	}{
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"warn", LevelWarn},
		{"error", LevelError},
		{"unknown", LevelInfo}, // default fallback
	}
	for _, c := range cases {
		if got := ParseLevel(c.in); got != c.want {
			t.Errorf("ParseLevel(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestLoadConfig_FromEnv(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	os.Setenv("LOG_WEBHOOK_URL", "http://example.com/webhook")
	os.Setenv("APP_NAME", "test-app")
	os.Setenv("ENV", "staging")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Level != LevelWarn {
		t.Errorf("expected LevelWarn, got %v", cfg.Level)
	}
	if !strings.Contains(cfg.WebhookURL, "example.com") {
		t.Errorf("unexpected webhook URL: %v", cfg.WebhookURL)
	}
	if cfg.AppName != "test-app" {
		t.Errorf("unexpected app name: %v", cfg.AppName)
	}
	if cfg.Environment != "staging" {
		t.Errorf("unexpected env: %v", cfg.Environment)
	}
}

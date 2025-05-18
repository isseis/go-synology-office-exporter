//go:build integration
// +build integration

package synology_drive_api

import (
	"os"
	"testing"
)

// TestMain sets up the environment for integration tests.
// These tests will run against a real Synology NAS device.
func TestMain(m *testing.M) {
	// Verify required environment variables are set
	requiredVars := []string{"SYNOLOGY_NAS_URL", "SYNOLOGY_NAS_USER", "SYNOLOGY_NAS_PASS"}
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			panic(v + " environment variable is required for integration tests")
		}
	}

	os.Exit(m.Run())
}

// ResetMockLogin resets the mock login state to 'not logged in'.
// Call this at the start of each test to avoid state leakage between tests.
func ResetMockLogin() {}

// getNasUrl returns the Synology NAS URL from environment variables
func getNasUrl() string {
	return os.Getenv("SYNOLOGY_NAS_URL")
}

// getNasUser returns the Synology NAS username from environment variables
func getNasUser() string {
	return os.Getenv("SYNOLOGY_NAS_USER")
}

// getNasPass returns the Synology NAS password from environment variables
func getNasPass() string {
	return os.Getenv("SYNOLOGY_NAS_PASS")
}

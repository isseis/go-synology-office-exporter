//go:build test
// +build test

package download_history

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOption configures how a test DownloadHistory is created.
type TestOption func(*testConfig)

type testConfig struct {
	path         string
	useTemp      bool
	initialState *state
	loadCallback func()
}

// WithTempDir configures the test to use a temporary directory.
// If filename is empty, a default name will be used.
func WithTempDir(filename string) TestOption {
	return func(c *testConfig) {
		c.useTemp = true
		if filename != "" {
			c.path = filename
		}
	}
}

// WithLoadCallback configures the test to use a callback function that is called after Load.
func WithLoadCallback(callback func()) TestOption {
	return func(c *testConfig) {
		c.loadCallback = callback
	}
}

// WithInitialState configures the test to start in a specific state.
func WithInitialState(initialState state) TestOption {
	return func(c *testConfig) {
		c.initialState = &initialState
	}
}

// TestDownloadHistory holds a DownloadHistory instance along with test-specific information.
type TestDownloadHistory struct {
	*DownloadHistory
	TempDir     string // Only set when using temporary directory
	HistoryFile string // Full path to the history file, if using filesystem
	cleanup     func() // Cleanup function to remove temporary resources
}

// Close cleans up any temporary resources created for testing.
// It should be called when the test is done, typically using defer.
func (t *TestDownloadHistory) Close() {
	if t.cleanup != nil {
		t.cleanup()
	}
}

// NewDownloadHistoryForTest creates a new test instance of DownloadHistory.
// By default, it operates in memory-only mode. Use WithTempDir to enable filesystem operations.
// The caller is responsible for calling Close() to clean up resources.
//
// Example:
//
//	th := NewDownloadHistoryForTest(t, map[string]DownloadItem{...}, WithTempDir("history.json"))
//	defer th.Close()
func NewDownloadHistoryForTest(t *testing.T, items map[string]DownloadItem, opts ...TestOption) *TestDownloadHistory {
	t.Helper()

	if items == nil {
		items = make(map[string]DownloadItem)
	}

	// Default config (memory-only)
	cfg := &testConfig{
		path:    "test-history.json",
		useTemp: false,
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	dh := &DownloadHistory{
		items:         make(map[string]DownloadItem),
		state:         stateReady, // Set to ready state for testing
		DownloadCount: counter{},
		SkippedCount:  counter{},
		IgnoredCount:  counter{},
		ErrorCount:    counter{},
	}

	result := &TestDownloadHistory{
		DownloadHistory: dh,
	}

	if cfg.initialState != nil {
		dh.state = *cfg.initialState
	}

	// Set up filesystem if needed
	if cfg.useTemp {
		tempDir, err := os.MkdirTemp("", "download-history-test-*")
		require.NoError(t, err, "failed to create temp dir")

		historyFile := filepath.Join(tempDir, cfg.path)
		dh.path = historyFile

		result.TempDir = tempDir
		result.HistoryFile = historyFile
		result.cleanup = func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("failed to remove temp dir %s: %v", tempDir, err)
			}
		}
	} else {
		// Memory-only mode
		dh.path = ""
	}

	if cfg.loadCallback != nil {
		dh.loadCallback = cfg.loadCallback
	}

	// Copy items
	maps.Copy(dh.items, items)

	return result
}

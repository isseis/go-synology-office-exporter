package synology_drive_exporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	download_history "github.com/isseis/go-synology-office-exporter/download_history"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExporter_removeFile(t *testing.T) {
	tests := []struct {
		name        string
		dryRun      bool
		createFile  bool // Whether to create the test file for this test case
		shouldExist bool // Whether the file should exist after the operation
		expectErr   bool
	}{
		{
			name:        "Remove existing file",
			dryRun:      false,
			createFile:  true,
			shouldExist: false,
			expectErr:   false,
		},
		{
			name:        "Dry run should not remove file",
			dryRun:      true,
			createFile:  true,
			shouldExist: true, // In dry run, file should still exist
			expectErr:   false,
		},
		{
			name:        "Non-existent file should not error",
			dryRun:      false,
			createFile:  false,
			shouldExist: false,
			expectErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "testfile.txt")

			// Create a test file if needed for this test case
			if tt.createFile {
				err := os.WriteFile(testFile, []byte("test content"), 0644)
				require.NoError(t, err, "Failed to create test file")
			}

			e := &Exporter{
				dryRun: tt.dryRun,
			}

			err := e.removeFile(testFile)

			if tt.expectErr {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Unexpected error")
			}

			_, err = os.Stat(testFile)
			if tt.shouldExist {
				assert.NoError(t, err, "File should still exist but doesn't")
			} else if !os.IsNotExist(err) {
				t.Errorf("File should not exist but does or other error: %v", err)
			}
		})
	}
}

func TestExporter_cleanupObsoleteFiles(t *testing.T) {
	t.Run("Normal cleanup", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create some test files
		files := []string{
			filepath.Join(tempDir, "obsolete1.txt"),
			filepath.Join(tempDir, "obsolete2.txt"),
			filepath.Join(tempDir, "keep1.txt"),
		}

		for _, f := range files {
			err := os.WriteFile(f, []byte("test"), 0644)
			require.NoError(t, err, "Failed to create test file")
		}

		// Create a history with some files using the test helper with temp dir
		th := download_history.NewDownloadHistoryForTest(t, map[string]download_history.DownloadItem{
			files[0]: {
				FileID:         "file1",
				Hash:           "hash1",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusLoaded, // Mark as loaded to be removed
			},
			files[1]: {
				FileID:         "file2",
				Hash:           "hash2",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusLoaded, // Mark as loaded to be removed
			},
			files[2]: {
				FileID:         "file3",
				Hash:           "hash3",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusDownloaded, // Should be kept
			},
		}, download_history.WithTempDir("history.json"))
		defer th.Close()
		history := th.DownloadHistory

		e := &Exporter{
			fs:     NewMockFileSystem(),
			dryRun: false,
		}

		// Save the history to transition to stateSaved state
		if err := history.Save(); err != nil {
			t.Fatalf("Failed to save history: %v", err)
		}

		stats := &ExportStats{}
		e.cleanupObsoleteFiles(history, stats)

		// Verify the stats were updated correctly
		assert.Equal(t, 2, stats.Removed, "Should have removed 2 files")
		assert.Equal(t, 0, stats.RemoveErrs, "Should have no remove errors")

		// Verify the files were actually removed
		for _, f := range files {
			_, err := os.Stat(f)
			if strings.Contains(f, "obsolete") {
				assert.True(t, os.IsNotExist(err), "Obsolete file should have been removed: %s", f)
			} else {
				assert.NoError(t, err, "Non-obsolete file should still exist: %s", f)
			}
		}
	})

	t.Run("Dry run should not remove files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create some test files
		files := []string{
			filepath.Join(tempDir, "obsolete1.txt"),
			filepath.Join(tempDir, "obsolete2.txt"),
		}

		for _, f := range files {
			err := os.WriteFile(f, []byte("test"), 0644)
			require.NoError(t, err, "Failed to create test file")
		}

		// Create a history with some files using the test helper with temp dir
		th := download_history.NewDownloadHistoryForTest(t, map[string]download_history.DownloadItem{
			filepath.Join(tempDir, "obsolete1.txt"): {
				FileID:         "file1",
				Hash:           "hash1",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusLoaded, // Mark as loaded to be removed
			},
			filepath.Join(tempDir, "obsolete2.txt"): {
				FileID:         "file2",
				Hash:           "hash2",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusLoaded, // Mark as loaded to be removed
			},
		}, download_history.WithTempDir("history.json"))
		defer th.Close()
		history := th.DownloadHistory

		e := &Exporter{
			fs:     NewMockFileSystem(),
			dryRun: true, // Enable dry run
		}

		// Save the history to transition to stateSaved state
		if err := history.Save(); err != nil {
			t.Fatalf("Failed to save history: %v", err)
		}

		stats := &ExportStats{}
		e.cleanupObsoleteFiles(history, stats)

		// Verify files are counted as removed in dry run mode
		assert.Equal(t, 2, stats.Removed, "Should count files as removed in dry run")
		assert.Equal(t, 0, stats.RemoveErrs, "Should have no remove errors")

		// Verify the files were not actually removed in dry run mode
		for _, f := range files {
			_, err := os.Stat(f)
			assert.NoError(t, err, "File should still exist in dry run: %s", f)
		}
	})

	t.Run("Skip cleanup on errors", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create some test files
		files := []string{
			filepath.Join(tempDir, "obsolete1.txt"),
		}

		for _, f := range files {
			err := os.WriteFile(f, []byte("test"), 0644)
			require.NoError(t, err, "Failed to create test file")
		}

		// Create a history with some files
		th := download_history.NewDownloadHistoryForTest(t, map[string]download_history.DownloadItem{
			filepath.Join(tempDir, "obsolete1.txt"): {
				FileID:         "file1",
				Hash:           "hash1",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusLoaded, // Mark as loaded to be removed
			},
		})
		history := th.DownloadHistory
		defer th.Close()

		e := &Exporter{
			fs:     NewMockFileSystem(),
			dryRun: false,
		}

		stats := &ExportStats{
			DownloadErrs: 1, // Simulate a previous error
		}

		e.cleanupObsoleteFiles(history, stats)

		// Verify no files were removed due to previous errors
		assert.Equal(t, 0, stats.Removed, "Should not have removed any files due to previous errors")
		assert.Equal(t, 0, stats.RemoveErrs, "Should have no remove errors")

		// Verify the file still exists
		_, err := os.Stat(filepath.Join(tempDir, "obsolete1.txt"))
		assert.NoError(t, err, "File should still exist when cleanup is skipped due to errors")
	})
}

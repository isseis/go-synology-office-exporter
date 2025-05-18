//go:build !integration
// +build !integration

package download_history

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

func TestDownloadHistoryBasic(t *testing.T) {
	history, err := NewDownloadHistory("test_history.json")
	if err != nil {
		t.Fatalf("failed to create history: %v", err)
	}

	item := DownloadItem{
		FileID:         synd.FileID("id1"),
		Hash:           synd.FileHash("hash1"),
		DownloadTime:   time.Now(),
		DownloadStatus: StatusLoaded,
	}

	history.Items["file1"] = item
	got, ok := history.Items["file1"]
	if !ok || got.FileID != item.FileID {
		t.Errorf("SetItem or Items failed: got %+v", got)
	}
}

func TestDownloadHistoryStatusMethods(t *testing.T) {
	baseTime := time.Now().Truncate(time.Second)
	itemLoaded := DownloadItem{
		FileID:         "id1",
		Hash:           "hash1",
		DownloadTime:   baseTime,
		DownloadStatus: StatusLoaded,
	}
	itemOther := DownloadItem{
		FileID:         "id2",
		Hash:           "hash2",
		DownloadTime:   baseTime,
		DownloadStatus: StatusDownloaded,
	}

	t.Run("MarkSkipped - success", func(t *testing.T) {
		h, _ := NewDownloadHistory("dummy.json")
		h.Items["file1"] = itemLoaded
		err := h.MarkSkipped("file1")
		assert.NoError(t, err)
		assert.Equal(t, StatusSkipped, h.Items["file1"].DownloadStatus)
	})
	t.Run("MarkSkipped - not found", func(t *testing.T) {
		h, _ := NewDownloadHistory("dummy.json")
		err := h.MarkSkipped("notfound")
		assert.ErrorIs(t, err, ErrHistoryItemNotFound)
	})
	t.Run("MarkSkipped - wrong status", func(t *testing.T) {
		h, _ := NewDownloadHistory("dummy.json")
		h.Items["file1"] = itemOther
		err := h.MarkSkipped("file1")
		assert.ErrorIs(t, err, ErrHistoryInvalidStatus)
	})

	t.Run("SetDownloaded - update loaded", func(t *testing.T) {
		h, _ := NewDownloadHistory("dummy.json")
		h.Items["file2"] = itemLoaded
		newItem := itemLoaded
		newItem.FileID = "id3"
		newItem.Hash = "hash3"
		newItem.DownloadTime = baseTime.Add(time.Hour)
		err := h.SetDownloaded("file2", newItem)
		assert.NoError(t, err)
		assert.Equal(t, StatusDownloaded, h.Items["file2"].DownloadStatus)
		assert.Equal(t, "id3", string(h.Items["file2"].FileID))
		assert.Equal(t, "hash3", string(h.Items["file2"].Hash))
		assert.Equal(t, baseTime.Add(time.Hour), h.Items["file2"].DownloadTime)
	})
	t.Run("SetDownloaded - add new", func(t *testing.T) {
		h, _ := NewDownloadHistory("dummy.json")
		item := itemLoaded
		item.DownloadStatus = StatusLoaded
		err := h.SetDownloaded("file3", item)
		assert.NoError(t, err)
		assert.Equal(t, StatusDownloaded, h.Items["file3"].DownloadStatus)
	})
	t.Run("SetDownloaded - error on wrong status", func(t *testing.T) {
		h, _ := NewDownloadHistory("dummy.json")
		h.Items["file2"] = itemOther
		newItem := itemOther
		err := h.SetDownloaded("file2", newItem)
		assert.ErrorIs(t, err, ErrHistoryInvalidStatus)
	})
}

func TestNewDownloadHistory(t *testing.T) {
	t.Run("Valid filename", func(t *testing.T) {
		validNames := []string{
			"history.json",
			"/tmp/history.json",
			"./history.json",
			"../history.json",
			"path/to/history.json",
		}

		for _, name := range validNames {
			history, err := NewDownloadHistory(name)
			assert.NoError(t, err, "Expected no error for valid filename: "+name)
			assert.NotNil(t, history)
			assert.NotNil(t, history.Items)
		}
	})

	t.Run("Empty filename", func(t *testing.T) {
		history, err := NewDownloadHistory("")
		assert.Error(t, err)
		assert.Nil(t, history)
		assert.Contains(t, err.Error(), "filename cannot be empty")
	})

	t.Run("Invalid filename", func(t *testing.T) {
		invalidNames := []string{
			".",
			"..",
			"/",
			"/tmp/",
		}

		for _, name := range invalidNames {
			history, err := NewDownloadHistory(name)
			assert.Error(t, err, "Expected error for invalid filename: "+name)
			assert.Nil(t, history)
			assert.Contains(t, err.Error(), "invalid filename")
		}
	})
}
func TestLoadFromReader(t *testing.T) {
	history, err := NewDownloadHistory("dummy.json")
	require.NoError(t, err)

	json := `{
		"header": {
			"version": 2,
			"magic": "SYNOLOGY_OFFICE_EXPORTER",
			"created": "2023-10-01T12:34:56Z"
		},
		"items": [
			{
				"location": "/path/to/file.odoc",
				"file_id": "882614125167948399",
				"hash": "1234567890abcdef",
				"download_time": "2023-10-01T12:34:56Z"
			}
		]
	}`
	// Use test-only API for loading from custom reader
	err = history.loadFromReader(strings.NewReader(json))
	require.NoError(t, err)

	item := history.Items["/path/to/file.odoc"]
	require.NotNil(t, item)
	assert.Equal(t, "882614125167948399", string(item.FileID))
	assert.Equal(t, "1234567890abcdef", string(item.Hash))
	assert.Equal(t, time.Date(2023, 10, 1, 12, 34, 56, 0, time.UTC), item.DownloadTime)

	// Test for invalid JSON syntax
	t.Run("Invalid JSON syntax", func(t *testing.T) {
		invalidJSON := `{
			"header": {
				"version": 2,
				"magic": "SYNOLOGY_OFFICE_EXPORTER",
				"created": "2023-10-01T12:00:00Z"
			},
			"items": [
				{
					"location": "/path/to/file.odoc",
					"file_id": "882614125167948399",
					"hash": "1234567890abcdef",
					"download_time": "2023-10-01T12:00:00Z"
				}
			]
		` // Missing closing bracket
		// Use test-only API for loading from custom reader
		err = history.loadFromReader(strings.NewReader(invalidJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
	})

	// Test for invalid version
	t.Run("Invalid version", func(t *testing.T) {
		invalidVersionJSON := `{
			"header": {
				"version": 1,
				"magic": "SYNOLOGY_OFFICE_EXPORTER",
				"created": "2023-10-01T12:00:00Z"
			},
			"items": []
		}`
		// Use test-only API for loading from custom reader
		err = history.loadFromReader(strings.NewReader(invalidVersionJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unsupported version: 1")
	})

	// Test for invalid magic string
	t.Run("Invalid magic string", func(t *testing.T) {
		invalidMagicJSON := `{
			"header": {
				"version": 2,
				"magic": "WRONG_MAGIC_STRING",
				"created": "2023-10-01T12:00:00Z"
			},
			"items": []
		}`
		// Use test-only API for loading from custom reader
		err = history.loadFromReader(strings.NewReader(invalidMagicJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "invalid magic: WRONG_MAGIC_STRING")
	})

	// Test for invalid date format
	t.Run("Invalid date format", func(t *testing.T) {
		invalidDateJSON := `{
			"header": {
				"version": 2,
				"magic": "SYNOLOGY_OFFICE_EXPORTER",
				"created": "2023-10-01T12:00:00Z"
			},
			"items": [
				{
					"location": "/path/to/file.odoc",
					"file_id": "882614125167948399",
					"hash": "1234567890abcdef",
					"download_time": "2023-10-01 12:00:00"
				}
			]
		}`
		// Use test-only API for loading from custom reader
		err = history.loadFromReader(strings.NewReader(invalidDateJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to parse download time")
	})

	// Test for duplicate locations
	t.Run("Duplicate locations", func(t *testing.T) {
		duplicateLocationJSON := `{
			"header": {
				"version": 2,
				"magic": "SYNOLOGY_OFFICE_EXPORTER",
				"created": "2023-10-01T12:00:00Z"
			},
			"items": [
				{
					"location": "/path/to/file.odoc",
					"file_id": "882614125167948399",
					"hash": "1234567890abcdef",
					"download_time": "2023-10-01T12:00:00Z"
				},
				{
					"location": "/path/to/file.odoc",
					"file_id": "882614125167948400",
					"hash": "1234567890abcdef",
					"download_time": "2023-10-01T13:00:00Z"
				}
			]
		}`
		// Use test-only API for loading from custom reader
		err = history.loadFromReader(strings.NewReader(duplicateLocationJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "duplicate location")
	})
}

func TestLoad(t *testing.T) {
	// Create a JSON file for testing in a temporary directory
	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "history.json")

	validJSON := `{
		"header": {
			"version": 2,
			"magic": "SYNOLOGY_OFFICE_EXPORTER",
			"created": "2023-10-01T15:27:38Z"
		},
		"items": [
			{
				"location": "/path/to/file.odoc",
				"file_id": "882614125167948399",
				"hash": "1234567890abcdef",
				"download_time": "2023-10-01T15:27:38Z"
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(validJSON), 0644)
	assert.Nil(t, err)

	// Test for valid file loading
	t.Run("Valid file", func(t *testing.T) {
		history, err := NewDownloadHistory(jsonPath)
		require.NoError(t, err)

		err = history.Load()
		assert.Nil(t, err)
		assert.Len(t, history.Items, 1)
		assert.Contains(t, history.Items, "/path/to/file.odoc")

		// Verify that the values are loaded correctly
		item, exists := history.Items["/path/to/file.odoc"]
		assert.True(t, exists)
		assert.Equal(t, "882614125167948399", string(item.FileID))
		assert.Equal(t, "1234567890abcdef", string(item.Hash))
		// Check non-zero minutes and seconds in timestamp
		assert.Equal(t, time.Date(2023, 10, 1, 15, 27, 38, 0, time.UTC), item.DownloadTime)
		assert.Equal(t, 27, item.DownloadTime.Minute())
		assert.Equal(t, 38, item.DownloadTime.Second())
	})

	// Test for non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "non_existent.json")
		history, err := NewDownloadHistory(nonExistentPath)
		require.NoError(t, err)

		err = history.Load()
		assert.NoError(t, err)
	})
}

// TestSaveToWriter tests SaveToWriter for successful and error cases, including output validation and error propagation.
func TestSaveToWriter(t *testing.T) {
	// Create a history instance with test data for testing
	history, err := NewDownloadHistory("dummy.json")
	require.NoError(t, err)

	history.Items = map[string]DownloadItem{
		"/path/to/file1.odoc": {
			FileID:       "882614125167948399",
			Hash:         "1234567890abcdef",
			DownloadTime: time.Date(2023, 10, 1, 12, 45, 23, 0, time.UTC),
		},
		"/path/to/file2.odoc": {
			FileID:       "882614125167948400",
			Hash:         "abcdef1234567890",
			DownloadTime: time.Date(2023, 10, 2, 8, 17, 39, 0, time.UTC),
		},
	}

	// Test successful case
	t.Run("Successful write", func(t *testing.T) {
		var buf strings.Builder
		// Use test-only API for saving to custom writer
		err = history.saveToWriter(&buf)
		assert.Nil(t, err)

		// Verify the output contains expected data
		output := buf.String()
		assert.Contains(t, output, HISTORY_MAGIC)
		assert.Contains(t, output, "\"version\": 2")
		assert.Contains(t, output, "/path/to/file1.odoc")
		assert.Contains(t, output, "/path/to/file2.odoc")
		assert.Contains(t, output, "882614125167948399")
		assert.Contains(t, output, "882614125167948400")
		assert.Contains(t, output, "1234567890abcdef")
		assert.Contains(t, output, "abcdef1234567890")
		assert.Contains(t, output, "2023-10-01T12:45:23Z")
		assert.Contains(t, output, "2023-10-02T08:17:39Z")

		// Parse the saved data to ensure it's valid
		loadedHistory, err := NewDownloadHistory("dummy.json")
		require.NoError(t, err)

		// Use test-only API for loading from custom reader
		err = loadedHistory.loadFromReader(strings.NewReader(output))
		assert.Nil(t, err)
		assert.Len(t, loadedHistory.Items, 2)

		// Verify timestamps with non-zero minutes and seconds are preserved
		file1, exists := loadedHistory.Items["/path/to/file1.odoc"]
		assert.True(t, exists)
		assert.Equal(t, 45, file1.DownloadTime.Minute())
		assert.Equal(t, 23, file1.DownloadTime.Second())

		file2, exists := loadedHistory.Items["/path/to/file2.odoc"]
		assert.True(t, exists)
		assert.Equal(t, 17, file2.DownloadTime.Minute())
		assert.Equal(t, 39, file2.DownloadTime.Second())
	})

	// Test error writing to the writer
	t.Run("Writer error", func(t *testing.T) {
		// Create a mock writer that returns an error on Write
		errorWriter := &mockErrorWriter{}
		// Use test-only API for saving to custom writer
		err = history.saveToWriter(errorWriter)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "file write error")
	})
}

// TestSave tests the Save method for successful file creation, content validation, and error scenarios such as file creation failure or permission issues.
func TestSave(t *testing.T) {
	// Test successful case
	t.Run("Successful save", func(t *testing.T) {
		tempDir := t.TempDir()
		jsonPath := filepath.Join(tempDir, "history.json")

		// Create a history instance with test data
		history, err := NewDownloadHistory(jsonPath)
		require.NoError(t, err)

		history.Items = map[string]DownloadItem{
			"/path/to/file.odoc": {
				FileID:       "882614125167948399",
				Hash:         "1234567890abcdef",
				DownloadTime: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
		}

		err = history.Save()
		assert.Nil(t, err)

		// Verify the file was created
		_, err = os.Stat(jsonPath)
		assert.Nil(t, err)

		// Read the file and verify its content
		data, err := os.ReadFile(jsonPath)
		assert.Nil(t, err)
		assert.Contains(t, string(data), HISTORY_MAGIC)
		assert.Contains(t, string(data), "/path/to/file.odoc")

		// Try loading the file to ensure it's valid
		loadedHistory, err := NewDownloadHistory(jsonPath)
		require.NoError(t, err)

		err = loadedHistory.Load()
		assert.Nil(t, err)
		assert.Len(t, loadedHistory.Items, 1)
	})

	// Test case when file creation fails
	t.Run("File creation error", func(t *testing.T) {
		// Try to save to a directory that doesn't exist
		history, err := NewDownloadHistory("/non-existent-dir/history.json")
		require.NoError(t, err)

		history.Items = map[string]DownloadItem{
			"/path/to/file.odoc": {
				FileID:       "882614125167948399",
				Hash:         "1234567890abcdef",
				DownloadTime: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
		}
		err = history.Save()
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "file write error")
	})

	// Test case when directory is not writable
	t.Run("Permission error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tempDir := t.TempDir()
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.Mkdir(readOnlyDir, 0500) // read-only directory
		assert.Nil(t, err)

		jsonPath := filepath.Join(readOnlyDir, "history.json")
		history, err := NewDownloadHistory(jsonPath)
		require.NoError(t, err)

		history.Items = map[string]DownloadItem{
			"/path/to/file.odoc": {
				FileID:       "882614125167948399",
				Hash:         "1234567890abcdef",
				DownloadTime: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
		}
		err = history.Save()
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "file write error")
	})
}

// mockErrorWriter is a test helper that implements io.Writer and always returns an error on Write.
type mockErrorWriter struct{}

func (m *mockErrorWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("mock write error")
}

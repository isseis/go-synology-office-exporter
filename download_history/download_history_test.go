//go:build test
// +build test

package download_history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

func TestDownloadHistoryBasic(t *testing.T) {
	item := DownloadItem{
		FileID:         synd.FileID("id1"),
		Hash:           synd.FileHash("hash1"),
		DownloadTime:   time.Now(),
		DownloadStatus: StatusLoaded,
	}

	history := NewDownloadHistoryForTest(t, map[string]DownloadItem{"file1": item})
	defer history.Close()

	got, exists, err := history.GetItem("file1")
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}
	if !exists || got.FileID != item.FileID {
		t.Errorf("GetItem failed: got %+v, exists=%v, error=%v", got, exists, err)
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
		th := NewDownloadHistoryForTest(t, map[string]DownloadItem{"file1": itemLoaded})
		defer th.Close()
		err := th.MarkSkipped("file1")
		assert.NoError(t, err)
		item, exists, err := th.GetItem("file1")
		require.NoError(t, err)
		require.True(t, exists)
		assert.Equal(t, StatusSkipped, item.DownloadStatus)
	})
	t.Run("MarkSkipped - not found", func(t *testing.T) {
		th := NewDownloadHistoryForTest(t, map[string]DownloadItem{})
		defer th.Close()
		err := th.MarkSkipped("notfound")
		assert.ErrorIs(t, err, ErrHistoryItemNotFound)
	})
	t.Run("MarkSkipped - wrong status", func(t *testing.T) {
		th := NewDownloadHistoryForTest(t, map[string]DownloadItem{"file1": itemOther})
		defer th.Close()
		err := th.MarkSkipped("file1")
		assert.ErrorIs(t, err, ErrHistoryInvalidStatus)
		item, exists, err := th.GetItem("file1")
		require.NoError(t, err)
		require.True(t, exists)
		assert.Equal(t, StatusDownloaded, item.DownloadStatus)
	})

	t.Run("SetDownloaded - update loaded", func(t *testing.T) {
		th := NewDownloadHistoryForTest(t, map[string]DownloadItem{"file2": itemLoaded})
		defer th.Close()
		newItem := itemLoaded
		newItem.FileID = "id3"
		newItem.Hash = "hash3"
		newItem.DownloadTime = baseTime.Add(time.Hour)
		err := th.SetDownloaded("file2", newItem)
		assert.NoError(t, err)
		item, exists, err := th.GetItem("file2")
		require.NoError(t, err)
		require.True(t, exists)
		assert.Equal(t, StatusDownloaded, item.DownloadStatus)
		assert.Equal(t, "id3", string(item.FileID))
		assert.Equal(t, "hash3", string(item.Hash))
		assert.Equal(t, baseTime.Add(time.Hour), item.DownloadTime)
	})
	t.Run("SetDownloaded - add new", func(t *testing.T) {
		th := NewDownloadHistoryForTest(t, map[string]DownloadItem{})
		defer th.Close()
		item := itemLoaded
		err := th.SetDownloaded("file3", item)
		assert.NoError(t, err)
		item, exists, err := th.GetItem("file3")
		require.NoError(t, err)
		require.True(t, exists)
		assert.Equal(t, StatusDownloaded, item.DownloadStatus)
	})
	t.Run("SetDownloaded - error on wrong status", func(t *testing.T) {
		th := NewDownloadHistoryForTest(t, map[string]DownloadItem{"file2": itemOther})
		defer th.Close()
		newItem := itemOther
		err := th.SetDownloaded("file2", newItem)
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
			assert.NotNil(t, history.items)
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
		assert.Len(t, history.items, 1)
		assert.Contains(t, history.items, "/path/to/file.odoc")

		// Verify that the values are loaded correctly
		item, exists := history.items["/path/to/file.odoc"]
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

// TestSave tests the Save method for successful file creation, content validation, and error scenarios such as file creation failure or permission issues.
func TestSave(t *testing.T) {
	// Test successful case
	t.Run("Successful save", func(t *testing.T) {
		tempDir := t.TempDir()
		jsonPath := filepath.Join(tempDir, "history.json")

		// Create a history instance with test data
		history, err := NewDownloadHistory(jsonPath)
		require.NoError(t, err)

		err = history.Load()
		require.NoError(t, err)

		// Add test data after Load()
		if history.items == nil {
			history.items = make(map[string]DownloadItem)
		}
		history.items["/path/to/file.odoc"] = DownloadItem{
			FileID:       "882614125167948399",
			Hash:         "1234567890abcdef",
			DownloadTime: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
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
		assert.Len(t, loadedHistory.items, 1)
	})

	// Test case when file creation fails
	t.Run("File creation error", func(t *testing.T) {
		// Create history with test data for non-existent directory
		history, err := NewDownloadHistory("/non-existent-dir/history.json")
		require.NoError(t, err)

		err = history.Load()
		require.NoError(t, err)

		// Add test data after Load()
		if history.items == nil {
			history.items = make(map[string]DownloadItem)
		}
		history.items["/path/to/file.odoc"] = DownloadItem{
			FileID:       "882614125167948399",
			Hash:         "1234567890abcdef",
			DownloadTime: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
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
		err = history.Load()
		require.NoError(t, err)

		// Add test data after Load()
		history.items = map[string]DownloadItem{
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

func TestGetObsoleteItems(t *testing.T) {
	baseTime := time.Now().Truncate(time.Second)
	item1 := DownloadItem{
		FileID:         "id1",
		Hash:           "hash1",
		DownloadTime:   baseTime,
		DownloadStatus: StatusLoaded,
	}
	item2 := DownloadItem{
		FileID:         "id2",
		Hash:           "hash2",
		DownloadTime:   baseTime,
		DownloadStatus: StatusLoaded,
	}
	item3 := DownloadItem{
		FileID:         "id3",
		Hash:           "hash3",
		DownloadTime:   baseTime,
		DownloadStatus: StatusLoaded,
	}

	t.Run("returns error when not saved", func(t *testing.T) {
		th := NewDownloadHistoryForTest(t, map[string]DownloadItem{
			"file1": item1,
			"file2": item2,
		})
		defer th.Close()

		// Should fail before Save() is called
		items, err := th.GetObsoleteItems()
		assert.ErrorIs(t, err, ErrNotReady)
		assert.Nil(t, items)
	})

	t.Run("returns unprocessed items after save", func(t *testing.T) {
		th := NewDownloadHistoryForTest(t, map[string]DownloadItem{
			"file1": item1,
			"file2": item2,
			"file3": item3,
		}, WithTempDir("history.json"))
		defer th.Close()

		// Process some items
		require.NoError(t, th.MarkSkipped("file1"))
		require.NoError(t, th.SetDownloaded("file2", DownloadItem{
			FileID:         "id2",
			Hash:           "hash2_updated",
			DownloadTime:   baseTime.Add(time.Hour),
			DownloadStatus: StatusDownloaded,
		}))

		// Save to move to stateSaved
		err := th.Save()
		require.NoError(t, err)

		// Get obsolete items (file3 was not processed)
		items, err := th.GetObsoleteItems()
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "file3", items[0])
	})

	t.Run("returns empty slice when all items are processed", func(t *testing.T) {
		th := NewDownloadHistoryForTest(t, map[string]DownloadItem{
			"file1": item1,
			"file2": item2,
		}, WithTempDir("history.json"))
		defer th.Close()

		// Process all items
		require.NoError(t, th.MarkSkipped("file1"))
		require.NoError(t, th.SetDownloaded("file2", DownloadItem{
			FileID:         "id2",
			Hash:           "hash2_updated",
			DownloadTime:   baseTime.Add(time.Hour),
			DownloadStatus: StatusDownloaded,
		}))

		// Save to move to stateSaved
		err := th.Save()
		require.NoError(t, err)

		// Should return empty slice when no obsolete items
		items, err := th.GetObsoleteItems()
		require.NoError(t, err)
		assert.Empty(t, items)
	})
}

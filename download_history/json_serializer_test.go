package download_history

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockErrorWriter struct{}

func (m *mockErrorWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("mock write error")
}

func TestLoadFromReader(t *testing.T) {
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

	items, err := loadItemsFromReader(strings.NewReader(json))
	require.NoError(t, err)

	item, exists := items["/path/to/file.odoc"]
	require.True(t, exists, "Expected item not found")
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

		_, err := loadItemsFromReader(strings.NewReader(invalidJSON))
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

		_, err := loadItemsFromReader(strings.NewReader(invalidVersionJSON))
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

		_, err := loadItemsFromReader(strings.NewReader(invalidMagicJSON))
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

		_, err := loadItemsFromReader(strings.NewReader(invalidDateJSON))
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

		_, err := loadItemsFromReader(strings.NewReader(duplicateLocationJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "duplicate location")
	})
}

// TestSaveToWriter tests SaveToWriter for successful and error cases, including output validation and error propagation.
func TestSaveToWriter(t *testing.T) {
	items := map[string]DownloadItem{
		"/path/to/file1.odoc": {
			FileID:         "882614125167948399",
			Hash:           "1234567890abcdef",
			DownloadTime:   time.Date(2023, 10, 1, 12, 45, 23, 0, time.UTC),
			DownloadStatus: StatusDownloaded,
		},
		"/path/to/file2.odoc": {
			FileID:         "882614125167948400",
			Hash:           "abcdef1234567890",
			DownloadTime:   time.Date(2023, 10, 2, 8, 17, 39, 0, time.UTC),
			DownloadStatus: StatusDownloaded,
		},
	}

	// Test successful case
	t.Run("Successful write", func(t *testing.T) {
		var buf strings.Builder
		// Save to custom writer using the package function
		err := saveToWriter(&buf, items)
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
		loadedItems, err := loadItemsFromReader(strings.NewReader(output))
		assert.Nil(t, err)
		assert.Len(t, loadedItems, 2)

		// Verify timestamps with non-zero minutes and seconds are preserved
		file1, exists := loadedItems["/path/to/file1.odoc"]
		assert.True(t, exists)
		assert.Equal(t, 45, file1.DownloadTime.Minute())
		assert.Equal(t, 23, file1.DownloadTime.Second())

		file2, exists := loadedItems["/path/to/file2.odoc"]
		assert.True(t, exists)
		assert.Equal(t, 17, file2.DownloadTime.Minute())
		assert.Equal(t, 39, file2.DownloadTime.Second())
	})

	// Test error writing to the writer
	t.Run("Writer error", func(t *testing.T) {
		// Create a mock writer that returns an error on Write
		errorWriter := &mockErrorWriter{}
		// Save to custom writer that returns an error
		err := saveToWriter(errorWriter, items)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "file write error")
	})
}

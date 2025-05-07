package synology_drive_exporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFromReader(t *testing.T) {
	history := NewDownloadHistory()
	json := `{
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
	}`
	err := history.loadFromReader(strings.NewReader(json))
	assert.Nil(t, err)

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
		err := history.loadFromReader(strings.NewReader(invalidJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to parse download history JSON")
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
		err := history.loadFromReader(strings.NewReader(invalidVersionJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to parse download history JSON")
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
		err := history.loadFromReader(strings.NewReader(invalidMagicJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to parse download history JSON")
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
		err := history.loadFromReader(strings.NewReader(invalidDateJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to parse download history JSON")
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
		err := history.loadFromReader(strings.NewReader(duplicateLocationJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to parse download history JSON")
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
	}`

	err := os.WriteFile(jsonPath, []byte(validJSON), 0644)
	assert.Nil(t, err)

	// Test for valid file loading
	t.Run("Valid file", func(t *testing.T) {
		history := NewDownloadHistory()
		err := history.Load(jsonPath)
		assert.Nil(t, err)
		assert.Len(t, history.Items, 1)
		assert.Contains(t, history.Items, "/path/to/file.odoc")
	})

	// Test for non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		history := NewDownloadHistory()
		err := history.Load(filepath.Join(tempDir, "non_existent.json"))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to read download history file")
	})
}

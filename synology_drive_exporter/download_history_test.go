package synology_drive_exporter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadHistory(t *testing.T) {

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
	err := history.Load(strings.NewReader(json))
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
		err := history.Load(strings.NewReader(invalidJSON))
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
		err := history.Load(strings.NewReader(invalidVersionJSON))
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
		err := history.Load(strings.NewReader(invalidMagicJSON))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to parse download history JSON")
		assert.Contains(t, err.Error(), "invalid magic: WRONG_MAGIC_STRING")
	})
}

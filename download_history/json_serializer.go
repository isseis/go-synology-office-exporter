package download_history

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

const (
	HISTORY_VERSION = 2
	HISTORY_MAGIC   = "SYNOLOGY_OFFICE_EXPORTER"
)

// jsonHeader is used for marshaling/unmarshaling metadata for download history JSON files.
type jsonHeader struct {
	Version int    `json:"version"`
	Magic   string `json:"magic"`
	Created string `json:"created"`
}

// jsonDownloadItem is used for marshaling/unmarshaling DownloadItem to/from JSON.
type jsonDownloadItem struct {
	Location     string        `json:"location"`
	FileID       synd.FileID   `json:"file_id"`
	Hash         synd.FileHash `json:"hash"`
	DownloadTime string        `json:"download_time"`
}

type jsonDownloadHistory struct {
	Header jsonHeader         `json:"header"`
	Items  []jsonDownloadItem `json:"items"`
}

// loadItemsFromReader loads items from a reader without holding any locks.
// It returns the loaded items and any error encountered.
func loadItemsFromReader(r io.Reader) (map[string]DownloadItem, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	var history jsonDownloadHistory
	if err := json.Unmarshal(content, &history); err != nil {
		return nil, fmt.Errorf("failed to decode history: %w", err)
	}

	if err := history.Header.validate(); err != nil {
		return nil, fmt.Errorf("invalid history header: %w", err)
	}

	items := make(map[string]DownloadItem, len(history.Items))
	for _, item := range history.Items {
		downloadTime, err := time.Parse(time.RFC3339, item.DownloadTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse download time: %w", err)
		}

		di := DownloadItem{
			FileID:         item.FileID,
			Hash:           item.Hash,
			DownloadTime:   downloadTime,
			DownloadStatus: StatusLoaded,
		}

		if _, exists := items[item.Location]; exists {
			return nil, fmt.Errorf("duplicate location: %s", item.Location)
		}

		items[item.Location] = di
	}

	return items, nil
}

// saveToWriter writes the provided items to the writer in JSON format.
// This function does not hold any locks and is safe to call with the items map.
func saveToWriter(w io.Writer, items map[string]DownloadItem) error {
	// Convert items to JSON structure
	jsonItems := make([]jsonDownloadItem, 0, len(items))
	for location, item := range items {
		jsonItems = append(jsonItems, jsonDownloadItem{
			Location:     location,
			FileID:       item.FileID,
			Hash:         item.Hash,
			DownloadTime: item.DownloadTime.Format(time.RFC3339),
		})
	}

	history := jsonDownloadHistory{
		Header: jsonHeader{
			Version: HISTORY_VERSION,
			Magic:   HISTORY_MAGIC,
			Created: time.Now().Format(time.RFC3339),
		},
		Items: jsonItems,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(history); err != nil {
		return fmt.Errorf("file write error: %w", err)
	}

	return nil
}

// validate checks that the JSON header matches the expected version and magic string.
func (hdr *jsonHeader) validate() error {
	if hdr.Version != HISTORY_VERSION {
		return fmt.Errorf("unsupported version: %d", hdr.Version)
	}
	if hdr.Magic != HISTORY_MAGIC {
		return fmt.Errorf("invalid magic: %s", hdr.Magic)
	}
	return nil
}

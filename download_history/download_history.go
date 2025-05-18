package download_history

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

const HISTORY_VERSION = 2
const HISTORY_MAGIC = "SYNOLOGY_OFFICE_EXPORTER"

type counter struct {
	count int
}

func (c *counter) Increment() {
	c.count++
}

func (c *counter) Get() int {
	return c.count
}

// ExportStats holds the statistics of the export operation.
type ExportStats struct {
	Downloaded int // Number of successfully downloaded files
	Skipped    int // Number of skipped files (already up-to-date)
	Ignored    int // Number of ignored files (not exportable)
	Errors     int // Number of errors occurred
}

// DownloadHistory manages the download state and statistics.
// DownloadHistory manages the download state and statistics.
type DownloadHistory struct {
	items map[string]DownloadItem // items holds the download history, private for encapsulation
	path  string

	DownloadCount counter
	SkippedCount  counter
	IgnoredCount  counter
	ErrorCount    counter
}

// SetItems replaces the internal items map (for testing only).
func (d *DownloadHistory) SetItems(m map[string]DownloadItem) {
	d.items = m
}

// MakeHistoryKey generates a key for download history from a display path.
// This replicates the logic previously used in synology_drive_exporter.MakeHistoryKey.
func MakeHistoryKey(displayPath string) string {
	return strings.TrimPrefix(filepath.Clean(displayPath), "/")
}

// ErrHistoryItemNotFound is returned when the specified item does not exist in the history.
var ErrHistoryItemNotFound = fmt.Errorf("download history item not found")

// ErrHistoryInvalidStatus is returned when the item's status does not match the expected state.
var ErrHistoryInvalidStatus = fmt.Errorf("download history item status is invalid")

// MarkSkipped sets the status of an existing item to 'skipped'.
// Returns an error if the item does not exist or if the item's status is not 'loaded'.
func (d *DownloadHistory) MarkSkipped(location string) error {
	item, ok := d.items[location]
	if !ok {
		return ErrHistoryItemNotFound
	}
	if item.DownloadStatus != StatusLoaded {
		return ErrHistoryInvalidStatus
	}
	item.DownloadStatus = StatusSkipped
	d.items[location] = item
	return nil
}

// SetDownloaded adds a new item with status 'downloaded' if it does not exist, or updates an existing item
// to 'downloaded' if its current status is 'loaded'. Returns an error if the item exists and its status is not 'loaded'.
func (d *DownloadHistory) SetDownloaded(location string, item DownloadItem) error {
	existing, ok := d.items[location]
	if ok {
		if existing.DownloadStatus != StatusLoaded {
			return ErrHistoryInvalidStatus
		}
		item.DownloadStatus = StatusDownloaded
		d.items[location] = item
		return nil
	}
	item.DownloadStatus = StatusDownloaded
	d.items[location] = item
	return nil
}

// GetStats returns the current export statistics.
func (d *DownloadHistory) GetStats() ExportStats {
	return ExportStats{
		Downloaded: d.DownloadCount.Get(),
		Skipped:    d.SkippedCount.Get(),
		Ignored:    d.IgnoredCount.Get(),
		Errors:     d.ErrorCount.Get(),
	}
}

// GetItem looks up a DownloadItem by its key.
// It returns the item and true if found, or false if not found.
func (d *DownloadHistory) GetItem(key string) (DownloadItem, bool) {
	item, exists := d.items[key]
	return item, exists
}

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

// DownloadStatus represents the state of a DownloadItem (enum-like string type).
type DownloadStatus string

const (
	StatusLoaded     DownloadStatus = "loaded"
	StatusDownloaded DownloadStatus = "downloaded"
	StatusSkipped    DownloadStatus = "skipped"
)

// DownloadItem holds information about a downloaded or tracked file, including its status.
type DownloadItem struct {
	FileID         synd.FileID
	Hash           synd.FileHash
	DownloadTime   time.Time
	DownloadStatus DownloadStatus
}

// NewDownloadHistory creates a new DownloadHistory instance with the specified path
// for later use with Save and Load methods.
//
// It validates that the path is not empty and could potentially be a valid file path.
// Returns an error if the filename is invalid.
func NewDownloadHistory(path string) (*DownloadHistory, error) {
	// Basic validity check
	if path == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	// Check for obviously invalid filenames
	if path == "." || path == ".." || path[len(path)-1] == '/' {
		return nil, fmt.Errorf("invalid filename: %s", path)
	}

	history := &DownloadHistory{
		items: make(map[string]DownloadItem),
		path:  path,
	}
	return history, nil
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

// loadFromReader loads DownloadItems from JSON and sets Status to "loaded" if not present.
// loadFromReader is a private helper that loads DownloadItems from JSON and sets Status to "loaded" if not present.
func (d *DownloadHistory) loadFromReader(r io.Reader) error {
	content, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("file read error: %s", err.Error())
	}

	var history jsonDownloadHistory
	if err := json.Unmarshal(content, &history); err != nil {
		return fmt.Errorf("parse error: %s", err.Error())
	}

	if err := history.Header.validate(); err != nil {
		return err
	}

	for _, item := range history.Items {
		downloadTime, err := time.Parse(time.RFC3339, item.DownloadTime)
		if err != nil {
			return fmt.Errorf("failed to parse download time: %s", err.Error())
		}
		// Prevent duplicate locations in the history map.
		if _, exists := d.items[item.Location]; exists {
			return fmt.Errorf("duplicate location: %s", item.Location)
		}
		// Set DownloadStatus to StatusLoaded on load to reflect state from file.
		d.items[item.Location] = DownloadItem{
			FileID:         item.FileID,
			Hash:           item.Hash,
			DownloadTime:   downloadTime,
			DownloadStatus: StatusLoaded,
		}
	}

	return nil
}

// Load reads download history from the JSON file specified.
// It returns an error if the file cannot be opened or contains invalid data.
func (d *DownloadHistory) Load() error {
	// If the file does not exist, treat as no history (not an error).
	if _, err := os.Stat(d.path); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(d.path)
	if err != nil {
		return fmt.Errorf("file read error: %s", err.Error())
	}
	defer file.Close()
	return d.loadFromReader(file)
}

// saveToWriter writes DownloadItems to JSON.
// saveToWriter is a private helper that writes DownloadItems to JSON.
func (d *DownloadHistory) saveToWriter(w io.Writer) error {
	header := jsonHeader{
		Version: HISTORY_VERSION,
		Magic:   HISTORY_MAGIC,
		Created: time.Now().Format(time.RFC3339),
	}

	items := make([]jsonDownloadItem, 0, len(d.items))
	for location, item := range d.items {
		items = append(items, jsonDownloadItem{
			Location:     location,
			FileID:       item.FileID,
			Hash:         item.Hash,
			DownloadTime: item.DownloadTime.Format(time.RFC3339),
		})
	}

	history := jsonDownloadHistory{
		Header: header,
		Items:  items,
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("file write error: %s", err.Error())
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("file write error: %s", err.Error())
	}
	return nil
}

// Save writes the download history to the JSON file specified during initialization.
// It returns an error if the file cannot be created or written to.
func (d *DownloadHistory) Save() error {
	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("file write error: %s", err.Error())
	}
	file, err := os.Create(d.path)
	if err != nil {
		return fmt.Errorf("file write error: %s", err.Error())
	}
	defer file.Close()
	return d.saveToWriter(file)
}

package synology_drive_exporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
type DownloadHistory struct {
	Items map[string]DownloadItem
	path  string

	DownloadCount counter
	SkippedCount  counter
	IgnoredCount  counter
	ErrorCount    counter
}

// ErrHistoryItemNotFound is returned when the specified item does not exist in the history.
var ErrHistoryItemNotFound = fmt.Errorf("download history item not found")

// ErrHistoryInvalidStatus is returned when the item's status does not match the expected state.
var ErrHistoryInvalidStatus = fmt.Errorf("download history item status is invalid")

// SetItem sets or updates a DownloadItem for the given location in the download history.
// This method should be used instead of direct access to the Items field from outside this struct.
func (d *DownloadHistory) SetItem(location string, item DownloadItem) {
	d.Items[location] = item
}

// MarkSkipped sets the status of an existing item to 'skipped' if its current status is 'loaded'.
// Returns an error if the item does not exist or its status is not 'loaded'.
func (d *DownloadHistory) MarkSkipped(location string) error {
	item, ok := d.Items[location]
	if !ok {
		return ErrHistoryItemNotFound
	}
	if item.DownloadStatus != StatusLoaded {
		return ErrHistoryInvalidStatus
	}
	item.DownloadStatus = StatusSkipped
	d.Items[location] = item
	return nil
}

// SetDownloaded adds a new item with status 'downloaded' if it does not exist, or updates an existing item
// to 'downloaded' if its current status is 'loaded'. Returns an error if the item exists and its status is not 'loaded'.
func (d *DownloadHistory) SetDownloaded(location string, item DownloadItem) error {
	existing, ok := d.Items[location]
	if ok {
		if existing.DownloadStatus != StatusLoaded {
			return ErrHistoryInvalidStatus
		}
		item.DownloadStatus = StatusDownloaded
		d.Items[location] = item
		return nil
	}
	item.DownloadStatus = StatusDownloaded
	d.Items[location] = item
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

/**
 * NewDownloadHistory creates a new DownloadHistory instance with the specified path
 * for later use with Save and Load methods.
 *
 * It validates that the path is not empty and could potentially be a valid file path.
 * Returns an error if the filename is invalid.
 */
func NewDownloadHistory(path string) (*DownloadHistory, error) {
	// Basic validity check
	if path == "" {
		return nil, DownloadHistoryFileError("filename cannot be empty")
	}

	// Check for obviously invalid filenames
	if path == "." || path == ".." || path[len(path)-1] == '/' {
		return nil, DownloadHistoryFileError(fmt.Sprintf("invalid filename: %s", path))
	}

	history := &DownloadHistory{
		Items: make(map[string]DownloadItem),
		path:  path,
	}
	return history, nil
}

func (hdr *jsonHeader) validate() error {
	if hdr.Version != HISTORY_VERSION {
		return DownloadHistoryParseError(fmt.Sprintf("unsupported version: %d", hdr.Version))
	}
	if hdr.Magic != HISTORY_MAGIC {
		return DownloadHistoryParseError(fmt.Sprintf("invalid magic: %s", hdr.Magic))
	}
	return nil
}

// loadFromReader loads DownloadItems from JSON and sets Status to "loaded" if not present.
func (d *DownloadHistory) loadFromReader(r io.Reader) error {
	content, err := io.ReadAll(r)
	if err != nil {
		return DownloadHistoryFileReadError(err.Error())
	}

	var history jsonDownloadHistory
	if err := json.Unmarshal(content, &history); err != nil {
		return DownloadHistoryParseError(err.Error())
	}

	if err := history.Header.validate(); err != nil {
		return err
	}

	for _, item := range history.Items {
		downloadTime, err := time.Parse(time.RFC3339, item.DownloadTime)
		if err != nil {
			return DownloadHistoryParseError(fmt.Sprintf("failed to parse download time: %s", err.Error()))
		}
		// Check if the location is already in the map
		if _, exists := d.Items[item.Location]; exists {
			return DownloadHistoryParseError(fmt.Sprintf("duplicate location: %s", item.Location))
		}
		// Always initialize Status as StatusLoaded on load
		d.Items[item.Location] = DownloadItem{
			FileID:         item.FileID,
			Hash:           item.Hash,
			DownloadTime:   downloadTime,
			DownloadStatus: StatusLoaded,
		}
	}

	return nil
}

// Load reads download history from the JSON file specified.
// It returns a DownloadHistoryFileError if the file cannot be opened
// or a DownloadHistoryParseError if the file contains invalid data.
func (d *DownloadHistory) Load() error {
	// If the file does not exist, we can just behave as if there is no history
	if _, err := os.Stat(d.path); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(d.path)
	if err != nil {
		return DownloadHistoryFileReadError(err.Error())
	}
	defer file.Close()
	return d.loadFromReader(file)
}

// saveToWriter writes DownloadItems to JSON, including their Status field.
func (d *DownloadHistory) saveToWriter(w io.Writer) error {
	header := jsonHeader{
		Version: HISTORY_VERSION,
		Magic:   HISTORY_MAGIC,
		Created: time.Now().Format(time.RFC3339),
	}

	items := make([]jsonDownloadItem, 0, len(d.Items))
	for location, item := range d.Items {
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
		return DownloadHistoryFileWriteError(err.Error())
	}

	if _, err := w.Write(data); err != nil {
		return DownloadHistoryFileWriteError(err.Error())
	}
	return nil
}

// Save writes the download history to the JSON file specified during initialization.
// It returns a DownloadHistoryFileError if the file cannot be created or written to.
// Save writes the download history to the JSON file specified during initialization.
// It returns a DownloadHistoryFileError if the file cannot be created or written to.
func (d *DownloadHistory) Save() error {
	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return DownloadHistoryFileWriteError(err.Error())
	}
	file, err := os.Create(d.path)
	if err != nil {
		return DownloadHistoryFileWriteError(err.Error())
	}
	defer file.Close()
	return d.saveToWriter(file)
}

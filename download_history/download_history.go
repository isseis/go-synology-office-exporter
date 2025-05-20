package download_history

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

const HISTORY_VERSION = 2
const HISTORY_MAGIC = "SYNOLOGY_OFFICE_EXPORTER"

type counter struct {
	count int32
}

func (c *counter) Increment() {
	atomic.AddInt32(&c.count, 1)
}

func (c *counter) Get() int {
	return int(atomic.LoadInt32(&c.count))
}

// ExportStats holds the statistics of the export operation.
type ExportStats struct {
	Downloaded int // Number of successfully downloaded files
	Skipped    int // Number of skipped files (already up-to-date)
	Ignored    int // Number of ignored files (not exportable)
	Errors     int // Number of errors occurred
}

// DownloadHistory manages the download state and statistics.
// All methods are safe for concurrent use.
type DownloadHistory struct {
	mu    sync.RWMutex
	items map[string]DownloadItem
	path  string

	// Counters are already thread-safe using atomic operations
	DownloadCount counter
	SkippedCount  counter
	IgnoredCount  counter
	ErrorCount    counter
}

// ErrHistoryItemNotFound is returned when the specified item does not exist in the history.
var ErrHistoryItemNotFound = fmt.Errorf("download history item not found")

// ErrHistoryInvalidStatus is returned when the item's status does not match the expected state.
var ErrHistoryInvalidStatus = fmt.Errorf("download history item status is invalid")

// MarkSkipped sets the status of an existing item to 'skipped'.
// Returns an error if the item does not exist or if the item's status is not 'loaded'.
// This method is safe for concurrent use.
func (d *DownloadHistory) MarkSkipped(location string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

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
// This method is safe for concurrent use.
func (d *DownloadHistory) SetDownloaded(location string, item DownloadItem) error {
	d.mu.Lock()
	defer d.mu.Unlock()

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

// GetItem looks up a DownloadItem by its key in a thread-safe manner.
// It returns the item and true if found, or false if not found.
func (d *DownloadHistory) GetItem(key string) (DownloadItem, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
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

// loadItemsFromReader loads items from a reader without holding any locks.
// It returns the loaded items and any error encountered.
func (d *DownloadHistory) loadItemsFromReader(r io.Reader) (map[string]DownloadItem, error) {
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

// loadFromReader loads DownloadItems from JSON and updates the internal state.
// It holds the lock only when updating the internal state.
func (d *DownloadHistory) loadFromReader(r io.Reader) error {
	items, err := d.loadItemsFromReader(r)
	if err != nil {
		return err
	}

	d.mu.Lock()
	d.items = items
	d.mu.Unlock()
	return nil
}

// Load reads download history from the JSON file specified during initialization.
// It returns an error if the file cannot be opened or contains invalid data.
// This method is safe for concurrent use.
func (d *DownloadHistory) Load() error {
	// First check if file exists without holding the lock
	if _, err := os.Stat(d.path); os.IsNotExist(err) {
		d.mu.Lock()
		d.items = make(map[string]DownloadItem)
		d.mu.Unlock()
		return nil
	}

	file, err := os.Open(d.path)
	if err != nil {
		return fmt.Errorf("file read error: %w", err)
	}
	defer file.Close()

	return d.loadFromReader(file)
}

// Save writes the download history to the JSON file specified during initialization.
// It returns an error if the file cannot be created or written to.
// This method is safe for concurrent use.
func (d *DownloadHistory) Save() error {
	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("file write error: %w", err)
	}

	file, err := os.Create(d.path)
	if err != nil {
		return fmt.Errorf("file write error: %w", err)
	}
	defer file.Close()

	return d.saveToWriter(file)
}

// saveToWriter writes the download history to the provided writer in JSON format.
// This method is safe for concurrent use.
func (d *DownloadHistory) saveToWriter(w io.Writer) error {
	d.mu.RLock()
	items := make([]jsonDownloadItem, 0, len(d.items))
	for location, item := range d.items {
		items = append(items, jsonDownloadItem{
			Location:     location,
			FileID:       item.FileID,
			Hash:         item.Hash,
			DownloadTime: item.DownloadTime.Format(time.RFC3339),
		})
	}
	d.mu.RUnlock()

	history := jsonDownloadHistory{
		Header: jsonHeader{
			Version: HISTORY_VERSION,
			Magic:   HISTORY_MAGIC,
			Created: time.Now().Format(time.RFC3339),
		},
		Items: items,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(history); err != nil {
		return fmt.Errorf("file write error: %w", err)
	}

	return nil
}

// GetObsoleteItems returns a slice of file paths that are marked as "loaded" in the history.
// These represent files that exist in history but were not found in the current export.
// This method is safe for concurrent use.
func (d *DownloadHistory) GetObsoleteItems() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var obsolete []string
	for path, item := range d.items {
		if item.DownloadStatus == StatusLoaded {
			obsolete = append(obsolete, path)
		}
	}
	return obsolete
}

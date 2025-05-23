package download_history

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

const (
	HISTORY_VERSION = 2
	HISTORY_MAGIC   = "SYNOLOGY_OFFICE_EXPORTER"
)

type state int

const (
	stateNew state = iota
	stateLoading
	stateReady
	stateSaved
)

var (
	// ErrAlreadyLoaded is returned when Load() is called more than once
	ErrAlreadyLoaded = errors.New("already loaded")

	// ErrNotReady is returned when an operation is attempted before Load()
	ErrNotReady = errors.New("history not loaded, call Load() first")

	// ErrAlreadyClosed is returned when Save() is called after Close()
	ErrAlreadyClosed = errors.New("history is already closed")
)

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
// It enforces a strict lifecycle: New -> Load -> (operations) -> Save/Close
// All methods are safe for concurrent use.
type DownloadHistory struct {
	mu           sync.RWMutex
	items        map[string]DownloadItem
	path         string
	state        state
	loadCallback func()

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
// Returns an error if the item does not exist, if the item's status is not 'loaded',
// or if the history is not in the ready state.
// This method is safe for concurrent use.
func (d *DownloadHistory) MarkSkipped(location string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != stateReady {
		return ErrNotReady
	}

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
// to 'downloaded' if its current status is 'loaded'. Returns an error if the item exists and its status is not 'loaded',
// or if the history is not in the ready state.
// This method is safe for concurrent use.
func (d *DownloadHistory) SetDownloaded(location string, item DownloadItem) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != stateReady {
		return ErrNotReady
	}

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
// Returns an error if the history is not in the ready state.
func (d *DownloadHistory) GetStats() (ExportStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.state != stateReady {
		return ExportStats{}, ErrNotReady
	}
	return ExportStats{
		Downloaded: d.DownloadCount.Get(),
		Skipped:    d.SkippedCount.Get(),
		Ignored:    d.IgnoredCount.Get(),
		Errors:     d.ErrorCount.Get(),
	}, nil
}

// GetItem looks up a DownloadItem by its key in a thread-safe manner.
// It returns the item and true if found, or false if not found.
// Returns an error if the history is not in the ready state.
func (d *DownloadHistory) GetItem(key string) (DownloadItem, bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.state != stateReady {
		return DownloadItem{}, false, ErrNotReady
	}

	item, exists := d.items[key]
	return item, exists, nil
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
// The returned DownloadHistory is in the 'new' state and must be initialized with Load()
// before any other operations can be performed.
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
		state: stateNew,
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

// Load reads download history from the JSON file specified during initialization.
// It returns an error if the file cannot be opened, contains invalid data,
// or if Load() has already been called.
// This method is safe for concurrent use.
func (d *DownloadHistory) Load() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != stateNew {
		return ErrAlreadyLoaded
	}

	d.state = stateLoading
	if d.loadCallback != nil {
		d.loadCallback()
	}

	// Check if file exists
	if _, err := os.Stat(d.path); os.IsNotExist(err) {
		d.items = make(map[string]DownloadItem)
		d.state = stateReady
		return nil
	}

	file, err := os.Open(d.path)
	if err != nil {
		d.state = stateNew // Reset state on error
		return fmt.Errorf("file read error: %w", err)
	}
	defer file.Close()

	// Load items in a separate function to ensure file is closed
	items, err := loadItemsFromReader(file)
	if err != nil {
		d.state = stateNew // Reset state on error
		return err
	}

	d.items = items
	d.state = stateReady
	return nil
}

// Save writes the download history to the JSON file specified during initialization.
// It returns an error if the file cannot be created or written to, or if the history
// is not in the ready state.
// This method is safe for concurrent use.
func (d *DownloadHistory) Save() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch d.state {
	case stateNew:
		return ErrNotReady
	case stateLoading:
		return ErrNotReady
	case stateSaved:
		return ErrAlreadyClosed
	}

	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("file write error: %w", err)
	}

	file, err := os.Create(d.path)
	if err != nil {
		return fmt.Errorf("file write error: %w", err)
	}
	defer file.Close()

	// Copy the items we want to save while holding the read lock
	items := make(map[string]DownloadItem, len(d.items))
	maps.Copy(items, d.items)

	if err := saveToWriter(file, items); err != nil {
		// Try to clean up the file if there was an error
		file.Close()
		os.Remove(d.path)
		return err
	}
	d.state = stateSaved
	return nil
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

// GetObsoleteItems returns a slice of file paths that are marked as "loaded" in the history.
// These represent files that exist in history but were not found in the current export.
// Returns an error if the history is not in the ready state.
// This method is safe for concurrent use.
func (d *DownloadHistory) GetObsoleteItems() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.state != stateSaved {
		return nil, ErrNotReady
	}

	var obsolete []string
	for path, item := range d.items {
		if item.DownloadStatus == StatusLoaded {
			obsolete = append(obsolete, path)
		}
	}
	return obsolete, nil
}

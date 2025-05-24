package download_history

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"
)

// state represents the lifecycle state of the DownloadHistory.
type state int

const (
	// stateNew is the initial state after NewDownloadHistory.
	stateNew state = iota
	// stateLoading is the state after Load() is called.
	stateLoading
	// stateReady is the state after Load() has successfully loaded the history.
	stateReady
	// stateSaved is the state after Save() has successfully saved the history.
	stateSaved
)

// DownloadHistory manages the download state and statistics.
// It enforces a strict lifecycle: New -> Load -> (operations) -> Save
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

var (
	// ErrAlreadyLoaded is returned when Load() is called more than once
	ErrAlreadyLoaded = errors.New("already loaded")

	// ErrNotReady is returned when an operation is attempted before Load()
	ErrNotReady = errors.New("history not loaded, call Load() first")

	// ErrAlreadyClosed is returned when Save() is called after Close()
	ErrAlreadyClosed = errors.New("history is already closed")

	// ErrHistoryItemNotFound is returned when the specified item does not exist in the history.
	ErrHistoryItemNotFound = fmt.Errorf("download history item not found")

	// ErrHistoryInvalidStatus is returned when the item's status does not match the expected state.
	ErrHistoryInvalidStatus = fmt.Errorf("download history item status is invalid")
)

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

// GetStats returns the current export statistics.
func (d *DownloadHistory) GetStats() ExportStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return ExportStats{
		Downloaded: d.DownloadCount.Get(),
		Skipped:    d.SkippedCount.Get(),
		Ignored:    d.IgnoredCount.Get(),
		Errors:     d.ErrorCount.Get(),
	}
}

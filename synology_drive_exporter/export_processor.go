package synology_drive_exporter

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	dh "github.com/isseis/go-synology-office-exporter/download_history"
	"github.com/isseis/go-synology-office-exporter/filelock"
	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// ExportItem represents an item to be exported.
type ExportItem struct {
	Type        synd.ObjectType
	FileID      synd.FileID
	DisplayPath string
	Hash        synd.FileHash
}

// newExportItem creates a new ExportItem from a ResponseItem.
func newExportItem(item *synd.ResponseItem) ExportItem {
	return ExportItem{
		Type:        item.Type,
		FileID:      item.FileID,
		DisplayPath: item.DisplayPath,
		Hash:        item.Hash,
	}
}

// makeLocalFileName generates a local file name from a display path.
func makeLocalFileName(displayPath string) string {
	return synd.GetExportFileName(strings.TrimPrefix(filepath.Clean(displayPath), "/"))
}

// processItem processes a single item (file or directory). Directories are processed recursively, exportable files are exported, and errors are logged. DownloadItem.Status distinguishes loaded, downloaded, and skipped states.
func (e *Exporter) processItem(item ExportItem, history *dh.DownloadHistory) {
	switch item.Type {
	case synd.ObjectTypeDirectory:
		e.processDirectory(item, history)
	case synd.ObjectTypeFile:
		e.processFile(item, history)
	}
}

// processFile exports a single convertible file and updates download history. Handles export, skip, and error logic.
// In dry-run mode, no file operations are performed; only statistics are updated.
// If forceDownload is true, files will be re-downloaded even if they exist and have matching hashes.
func (e *Exporter) processFile(item ExportItem, history *dh.DownloadHistory) {
	exportName := synd.GetExportFileName(item.DisplayPath)
	if exportName == "" {
		e.getLogger().Info("Skipping non-exportable file", "path", item.DisplayPath)
		history.IgnoredCount.Increment()
		return
	}

	localPath := makeLocalFileName(item.DisplayPath)

	// Check if we should skip based on hash and forceDownload flag
	prev, downloaded, err := history.GetItem(localPath)
	if err != nil {
		e.getLogger().Error("Failed to check download history", "path", localPath, "error", err)
		history.ErrorCount.Increment()
		return
	}

	// Skip if file exists, hashes match, and we're not forcing a re-download
	if !e.forceDownload && downloaded && prev.Hash == item.Hash {
		e.getLogger().Debug("Skipping file due to unchanged hash", "path", localPath, "prev_hash", prev.Hash, "current_hash", item.Hash, "status", prev.DownloadStatus)
		history.SkippedCount.Increment()
		err := history.MarkSkipped(localPath)
		if err != nil {
			e.getLogger().Warn("Failed to mark file as skipped in history", "path", localPath, "error", err)
		}
		return
	}

	// If we're forcing a download and the file exists, log that we're re-downloading
	if e.forceDownload && downloaded {
		e.getLogger().Info("Re-downloading file due to force-download option", "path", localPath)
	}
	if e.IsDryRun() {
		e.getLogger().Info("Dry run: would export file", "export_name", exportName)
		// Simulate successful export for statistics only
		newItem := dh.DownloadItem{
			FileID:       item.FileID,
			Hash:         item.Hash,
			DownloadTime: time.Now(),
		}
		errHistory := history.SetDownloaded(localPath, newItem)
		if errHistory != nil {
			e.getLogger().Warn("Failed to update download history in dry run", "path", localPath, "error", errHistory)
		}
		history.DownloadCount.Increment()
		return
	}
	e.getLogger().Info("Exporting file", "export_name", exportName)
	resp, err := e.session.Export(item.FileID)
	if err != nil {
		e.getLogger().Error("Failed to export file", "export_name", exportName, "error", err)
		history.ErrorCount.Increment()
		return
	}
	downloadPath := filepath.Join(e.downloadDir, localPath)
	if err := e.fs.CreateFile(downloadPath, resp.Content, 0755, 0644); err != nil {
		e.getLogger().Error("Failed to write file", "path", downloadPath, "error", err)
		history.ErrorCount.Increment()
		return
	}

	e.getLogger().Info("File exported successfully", "path", downloadPath)
	// Update download history: if entry exists, mark as downloaded (only if loaded); otherwise add as new downloaded entry.
	newItem := dh.DownloadItem{
		FileID:       item.FileID,
		Hash:         item.Hash,
		DownloadTime: time.Now(),
	}
	// SetDownloaded: add new entry or update existing (if loaded) to 'downloaded'.
	errHistory := history.SetDownloaded(localPath, newItem)
	if errHistory != nil {
		e.getLogger().Warn("Failed to update download history", "path", localPath, "error", errHistory)
	}
	history.DownloadCount.Increment()
}

// processDirectory recursively processes a directory and its subdirectories, exporting convertible files and recording errors in history.
func (e *Exporter) processDirectory(item ExportItem, history *dh.DownloadHistory) {
	// Use listAll to handle pagination automatically
	items, err := listAll(e.session, item.FileID)
	if err != nil {
		e.getLogger().Error("Failed to list directory", "path", item.DisplayPath, "error", err)
		history.ErrorCount.Increment()
		return
	}
	for _, child := range items {
		e.processItem(newExportItem(child), history)
	}
}

// exportItemsWithHistory is an internal helper for exporting a slice of ExportItem with download history management.
// Only one process can execute this function for a given history file at a time.
// If another process is already processing the same history file, this function will return an error.
func (e *Exporter) exportItemsWithHistory(
	items []ExportItem,
	historyFile string,
) (ExportStats, error) {
	historyPath := filepath.Join(e.downloadDir, historyFile)

	// Acquire a file lock to prevent concurrent execution for the same history file
	unlock, err := filelock.TryLock(historyPath)
	if err != nil {
		if errors.Is(err, filelock.ErrLockHeld) {
			return ExportStats{}, fmt.Errorf("another process is already exporting to %s", historyFile)
		}
		return ExportStats{}, fmt.Errorf("failed to acquire lock for %s: %w", historyFile, err)
	}
	defer unlock()

	history, err := dh.NewDownloadHistory(historyPath)
	if err != nil {
		return ExportStats{}, &DownloadHistoryOperationError{Op: "create", Err: err}
	}
	if err := history.Load(); err != nil {
		return ExportStats{}, &DownloadHistoryOperationError{Op: "load", Err: err}
	}
	for _, item := range items {
		e.processItem(item, history)
	}

	dlStats := history.GetStats()
	exStats := toExportStats(dlStats)
	if err := history.Save(); err != nil {
		return exStats, &DownloadHistoryOperationError{Op: "save", Err: err}
	}

	if err := e.cleanupObsoleteFiles(history, &exStats); err != nil {
		return exStats, &DownloadHistoryOperationError{Op: "cleanup obsolete files", Err: err}
	}
	return exStats, nil
}

// ExportRootsWithHistory exports multiple root directories with download history management.
func (e *Exporter) ExportRootsWithHistory(
	rootIDs []synd.FileID,
	historyFile string,
) (ExportStats, error) {
	var exportItems []ExportItem
	for _, rootID := range rootIDs {
		exportItems = append(exportItems, ExportItem{
			Type:        synd.ObjectTypeDirectory,
			FileID:      rootID,
			DisplayPath: "",
			Hash:        "",
		})
	}
	return e.exportItemsWithHistory(exportItems, historyFile)
}

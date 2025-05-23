package synology_drive_exporter

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/isseis/go-synology-office-exporter/download_history"
	"github.com/isseis/go-synology-office-exporter/filelock"
	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// ExportStats holds the statistics of the export operation
type ExportStats struct {
	Downloaded   int // Number of successfully downloaded files
	Skipped      int // Number of skipped files (already up-to-date)
	Ignored      int // Number of ignored files (not exportable)
	Removed      int // Number of successfully removed files
	DownloadErrs int // Number of errors occurred during download
	RemoveErrs   int // Number of errors occurred during removal
}

// String returns a string representation of the export statistics
func (s ExportStats) String() string {
	return fmt.Sprintf("downloaded=%d, skipped=%d, ignored=%d, removed=%d, download_errors=%d, remove_errors=%d",
		s.Downloaded, s.Skipped, s.Ignored, s.Removed, s.DownloadErrs, s.RemoveErrs)
}

func (s *ExportStats) IncrementRemoved() {
	s.Removed++
}

// IncrementDownloadErrs increments the download error count
func (s *ExportStats) IncrementDownloadErrs() {
	s.DownloadErrs++
}

// IncrementRemoveErrs increments the removal error count
func (s *ExportStats) IncrementRemoveErrs() {
	s.RemoveErrs++
}

// TotalErrs returns the total number of errors
func (s *ExportStats) TotalErrs() int {
	return s.DownloadErrs + s.RemoveErrs
}

// FileSystemOperations abstracts file system operations for the Exporter, allowing for easy mocking in tests.
type FileSystemOperations interface {
	// CreateFile writes data to a file, creating parent directories if needed. Directory and file permissions are set by dirPerm and filePerm.
	CreateFile(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error
}

// DefaultFileSystem provides a production implementation of FileSystemOperations using the os package.
type DefaultFileSystem struct{}

func (fs *DefaultFileSystem) CreateFile(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return ExportFileWriteError{Op: fmt.Sprintf("MkdirAll for %s", dir), Err: err}
	}

	// Write data to the file
	if err := os.WriteFile(filename, data, filePerm); err != nil {
		return ExportFileWriteError{Op: fmt.Sprintf("WriteFile for %s", filename), Err: err}
	}

	return nil
}

// Exporter handles exporting files from Synology Drive, maintaining download history and file system abstraction.
type Exporter struct {
	session     SessionInterface
	downloadDir string // Directory where downloaded files will be saved
	fs          FileSystemOperations

	// dryRun controls whether file operations are performed. Immutable after construction.
	// Default is false.
	dryRun bool

	// forceDownload controls whether to re-download files even if they exist and have matching hashes.
	// Default is false.
	forceDownload bool
}

// ExporterOption defines a function type to set options for Exporter.
// Use WithDryRun and similar helpers to specify runtime options.
type ExporterOption func(*Exporter)

// WithDryRun sets the dryRun option for Exporter.
func WithDryRun(dryRun bool) ExporterOption {
	return func(e *Exporter) {
		e.dryRun = dryRun
	}
}

// WithForceDownload sets the forceDownload option for Exporter.
// When true, files will be re-downloaded even if they exist and have matching hashes.
func WithForceDownload(force bool) ExporterOption {
	return func(e *Exporter) {
		e.forceDownload = force
	}
}

// IsDryRun returns true if the exporter is in dry-run mode.
func (e *Exporter) IsDryRun() bool {
	return e.dryRun
}

// NewExporter constructs an Exporter with a real Synology session and the specified download directory. If downloadDir is empty, the current directory is used.
// Additional runtime options can be specified via ExporterOption(s), such as WithDryRun.
func NewExporter(username string, password string, base_url string, downloadDir string, opts ...ExporterOption) (*Exporter, error) {
	session, err := synd.NewSynologySession(username, password, base_url)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	if err = session.Login(); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	exporter := NewExporterWithDependencies(session, downloadDir, &DefaultFileSystem{}, opts...)
	return exporter, nil
}

// NewExporterWithDependencies constructs an Exporter with injected dependencies for session, download directory, and file system. Intended for testing and advanced use.
// Additional runtime options can be specified via ExporterOption(s), such as WithDryRun.
func NewExporterWithDependencies(session SessionInterface, downloadDir string, fs FileSystemOperations, opts ...ExporterOption) *Exporter {
	e := &Exporter{
		session:     session,
		downloadDir: downloadDir,
		fs:          fs,
		dryRun:      false, // default
	}
	// Apply additional runtime options.
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// ExportMyDrive exports convertible files from the user's Synology Drive, using download history to avoid duplicates.
func (e *Exporter) ExportMyDrive() (ExportStats, error) {
	return e.ExportRootsWithHistory(
		[]synd.FileID{synd.MyDrive},
		"mydrive_history.json",
	)
}

// ExportTeamFolder exports convertible files from all team folders, using download history to avoid duplicates.
func (e *Exporter) ExportTeamFolder() (ExportStats, error) {
	teamFolders, err := teamFoldersAll(e.session)
	if err != nil {
		return ExportStats{}, err
	}
	var rootIDs []synd.FileID
	for _, item := range teamFolders {
		rootIDs = append(rootIDs, item.FileID)
	}
	return e.ExportRootsWithHistory(
		rootIDs,
		"team_folder_history.json",
	)
}

// ExportSharedWithMe exports convertible files and directories shared with the user, using download history to avoid duplicates.
func (e *Exporter) ExportSharedWithMe() (ExportStats, error) {
	sharedItems, err := sharedWithMeAll(e.session)
	if err != nil {
		return ExportStats{}, err
	}
	var exportItems []ExportItem
	for _, item := range sharedItems {
		exportItems = append(exportItems, toExportItem(item))
	}
	return e.exportItemsWithHistory(exportItems, "shared_with_me_history.json")
}

func toExportStats(stats download_history.ExportStats) ExportStats {
	return ExportStats{
		Downloaded:   stats.Downloaded,
		Skipped:      stats.Skipped,
		Ignored:      stats.Ignored,
		Removed:      0,
		DownloadErrs: stats.Errors,
		RemoveErrs:   0,
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

	history, err := download_history.NewDownloadHistory(historyPath)
	if err != nil {
		return ExportStats{}, &DownloadHistoryOperationError{Op: "create", Err: err}
	}
	if err := history.Load(); err != nil {
		return ExportStats{}, &DownloadHistoryOperationError{Op: "load", Err: err}
	}
	for _, item := range items {
		e.processItem(item, history)
	}

	var dlStats download_history.ExportStats
	if dlStats, err = history.GetStats(); err != nil {
		return ExportStats{}, &DownloadHistoryOperationError{Op: "get stats", Err: err}
	}
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

// processDirectory recursively processes a directory and its subdirectories, exporting convertible files and recording errors in history.
func (e *Exporter) processDirectory(item ExportItem, history *download_history.DownloadHistory) {
	// Use listAll to handle pagination automatically
	items, err := listAll(e.session, item.FileID)
	if err != nil {
		fmt.Printf("Failed to list directory %s: %v\n", item.DisplayPath, err)
		history.ErrorCount.Increment()
		return
	}
	for _, child := range items {
		e.processItem(toExportItem(child), history)
	}
}

// processFile exports a single convertible file and updates download history. Handles export, skip, and error logic.
// In dry-run mode, no file operations are performed; only statistics are updated.
// If forceDownload is true, files will be re-downloaded even if they exist and have matching hashes.
func (e *Exporter) processFile(item ExportItem, history *download_history.DownloadHistory) {
	exportName := synd.GetExportFileName(item.DisplayPath)
	if exportName == "" {
		fmt.Printf("Skip (not exportable): %s\n", item.DisplayPath)
		history.IgnoredCount.Increment()
		return
	}

	localPath := makeLocalFileName(item.DisplayPath)

	// Check if we should skip based on hash and forceDownload flag
	prev, downloaded, err := history.GetItem(localPath)
	if err != nil {
		fmt.Printf("Error checking history for %s: %v\n", localPath, err)
		history.ErrorCount.Increment()
		return
	}

	// Skip if file exists, hashes match, and we're not forcing a re-download
	if !e.forceDownload && downloaded && prev.Hash == item.Hash {
		fmt.Printf("[DEBUG] hash skip: localPath=%s, prev.Hash=%s, item.Hash=%s, prev.DownloadStatus=%s\n", localPath, prev.Hash, item.Hash, prev.DownloadStatus)
		fmt.Printf("Skip (already exported and hash unchanged): %s\n", localPath)
		history.SkippedCount.Increment()
		err := history.MarkSkipped(localPath)
		if err != nil {
			fmt.Printf("Warning: could not mark as skipped: %v\n", err)
		}
		return
	}

	// If we're forcing a download and the file exists, log that we're re-downloading
	if e.forceDownload && downloaded {
		fmt.Printf("Re-downloading (--force-download): %s\n", localPath)
	}
	if e.IsDryRun() {
		fmt.Printf("[DRY RUN] Would export file: %s\n", exportName)
		// Simulate successful export for statistics only
		newItem := download_history.DownloadItem{
			FileID:       item.FileID,
			Hash:         item.Hash,
			DownloadTime: time.Now(),
		}
		errHistory := history.SetDownloaded(localPath, newItem)
		if errHistory != nil {
			fmt.Printf("Warning: could not update download history: %v\n", errHistory)
		}
		history.DownloadCount.Increment()
		return
	}
	fmt.Printf("Exporting file: %s\n", exportName)
	resp, err := e.session.Export(item.FileID)
	if err != nil {
		fmt.Printf("failed to export %s: %v\n", exportName, err)
		history.ErrorCount.Increment()
		return
	}
	downloadPath := filepath.Join(e.downloadDir, localPath)
	if err := e.fs.CreateFile(downloadPath, resp.Content, 0755, 0644); err != nil {
		fmt.Printf("failed to write file %s: %v\n", downloadPath, err)
		history.ErrorCount.Increment()
		return
	}

	fmt.Printf("Saved to: %s\n", downloadPath)
	// Update download history: if entry exists, mark as downloaded (only if loaded); otherwise add as new downloaded entry.
	newItem := download_history.DownloadItem{
		FileID:       item.FileID,
		Hash:         item.Hash,
		DownloadTime: time.Now(),
	}
	// SetDownloaded: add new entry or update existing (if loaded) to 'downloaded'.
	errHistory := history.SetDownloaded(localPath, newItem)
	if errHistory != nil {
		fmt.Printf("Warning: could not update download history: %v\n", errHistory)
	}
	history.DownloadCount.Increment()
}

// ExportItem contains only the fields needed for export processing, reducing dependency on the full ResponseItem struct.
type ExportItem struct {
	Type        synd.ObjectType
	FileID      synd.FileID
	DisplayPath string
	Hash        synd.FileHash
}

// makeLocalFileName generates a local file name from a display path.
func makeLocalFileName(displayPath string) string {
	return synd.GetExportFileName(strings.TrimPrefix(filepath.Clean(displayPath), "/"))
}

func toExportItem(item *synd.ResponseItem) ExportItem {
	return ExportItem{
		Type:        item.Type,
		FileID:      item.FileID,
		DisplayPath: item.DisplayPath,
		Hash:        item.Hash,
	}
}

// processItem processes a single item (file or directory). Directories are processed recursively, exportable files are exported, and errors are logged. DownloadItem.Status distinguishes loaded, downloaded, and skipped states.
func (e *Exporter) processItem(item ExportItem, history *download_history.DownloadHistory) {
	switch item.Type {
	case synd.ObjectTypeDirectory:
		e.processDirectory(item, history)
	case synd.ObjectTypeFile:
		e.processFile(item, history)
	}
}

// removeFile removes the specified file from the filesystem.
// In DryRun mode, it only logs the operation without actually removing the file.
func (e *Exporter) removeFile(path string) error {
	if e.IsDryRun() {
		log.Printf("[DRY RUN] Would remove file: %s", path)
		return nil
	}

	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("File already removed: %s", path)
			return nil
		}
		return fmt.Errorf("failed to remove file %s: %w", path, err)
	}
	log.Printf("Removed file: %s", path)
	return nil
}

// cleanupObsoleteFiles removes files that exist in history but not in the current export
// It skips cleanup if there were any errors during the export process
func (e *Exporter) cleanupObsoleteFiles(history *download_history.DownloadHistory, stats *ExportStats) error {
	if stats.TotalErrs() > 0 {
		log.Println("Skipping cleanup due to previous errors")
		return nil
	}

	obsoletePaths, err := history.GetObsoleteItems()
	if err != nil {
		log.Printf("Error getting obsolete items: %v", err)
		return err
	}
	for _, path := range obsoletePaths {
		if err := e.removeFile(path); err != nil {
			stats.IncrementRemoveErrs() // Always count errors
			log.Printf("Error removing file: %v", err)
		} else {
			stats.IncrementRemoved() // Always count successful removals
		}
	}
	return nil
}

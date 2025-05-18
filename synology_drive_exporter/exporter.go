package synology_drive_exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// FileSystemOperations defines an interface for file system operations used by the Exporter.
// This interface simplifies testing by allowing file system operations to be mocked.
type FileSystemOperations interface {
	// CreateFile writes data to a file, creating parent directories if they don't exist.
	// The `dirPerm` argument specifies the permissions for directories (e.g., 0755 allows
	// the owner to read, write, and execute, while others can only read and execute).
	// The `filePerm` argument specifies the permissions for files (e.g., 0644 allows
	// the owner to read and write, while others can only read).
	CreateFile(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error
}

// DefaultFileSystem implements the FileSystemOperations interface using the os package.
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

// SessionInterface defines an interface for the Synology session operations.
// This interface allows the session to be mocked for testing.
type SessionInterface interface {
	// List retrieves a list of items from the specified root directory.
	List(rootDirID synd.FileID) (*synd.ListResponse, error)

	// Export exports the specified file with conversion.
	Export(fileID synd.FileID) (*synd.ExportResponse, error)

	// TeamFolder retrieves a list of team folders from the Synology Drive API.
	TeamFolder() (*synd.TeamFolderResponse, error)

	// SharedWithMe retrieves a list of files shared with the user.
	SharedWithMe() (*synd.SharedWithMeResponse, error)
}

// Exporter provides functionality to export files from Synology Drive.
type Exporter struct {
	session     SessionInterface
	downloadDir string // Directory where downloaded files will be saved
	fs          FileSystemOperations
}

// NewExporter creates a new Exporter with the specified download directory.
// If downloadDir is not provided, current directory will be used as default.
func NewExporter(username string, password string, base_url string, downloadDir string) (*Exporter, error) {
	session, err := synd.NewSynologySession(username, password, base_url)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	if err = session.Login(); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	exporter := NewExporterWithDependencies(session, downloadDir, &DefaultFileSystem{})
	return exporter, nil
}

// NewExporterWithDependencies creates a new Exporter with injected dependencies (session, downloadDir, and file system).
// This constructor allows for dependency injection, making it suitable for both production and testing.
func NewExporterWithDependencies(session SessionInterface, downloadDir string, fs FileSystemOperations) *Exporter {
	return &Exporter{
		session:     session,
		downloadDir: downloadDir,
		fs:          fs,
	}
}

// ExportMyDrive exports all convertible files from the user's Synology Drive and saves them to the download directory.
// Download history is used to avoid duplicate downloads.
func (e *Exporter) ExportMyDrive() (ExportStats, error) {
	return e.ExportRootsWithHistory(
		[]synd.FileID{synd.MyDrive},
		"mydrive_history.json",
	)
}

// ExportTeamFolder exports all convertible files from all team folders.
// Download history is used to avoid duplicate downloads.
func (e *Exporter) ExportTeamFolder() (ExportStats, error) {
	teamFolder, err := e.session.TeamFolder()
	if err != nil {
		return ExportStats{}, err
	}
	var rootIDs []synd.FileID
	for _, item := range teamFolder.Items {
		rootIDs = append(rootIDs, item.FileID)
	}
	return e.ExportRootsWithHistory(
		rootIDs,
		"team_folder_history.json",
	)
}

// ExportSharedWithMe exports all convertible files and directories shared with the user.
// Download history is used to avoid duplicate downloads.
func (e *Exporter) ExportSharedWithMe() (ExportStats, error) {
	sharedWithMe, err := e.session.SharedWithMe()
	if err != nil {
		return ExportStats{}, err
	}
	var exportItems []ExportItem
	for _, item := range sharedWithMe.Items {
		exportItems = append(exportItems, toExportItem(item))
	}
	return e.exportItemsWithHistory(exportItems, "shared_with_me_history.json")
}

// exportItemsWithHistory is a common internal helper for exporting a slice of ExportItem with download history management.
// It handles DownloadHistory creation, loading, saving, and calls processItem for each item.
// This function is used by both ExportRootsWithHistory and ExportSharedWithMe to avoid code duplication.
// Returns ExportStats and error (wrapped in DownloadHistoryOperationError if relevant).
func (e *Exporter) exportItemsWithHistory(
	items []ExportItem,
	historyFile string,
) (ExportStats, error) {
	historyPath := filepath.Join(e.downloadDir, historyFile)
	history, err := NewDownloadHistory(historyPath)
	if err != nil {
		return ExportStats{}, &DownloadHistoryOperationError{Op: "create", Err: err}
	}
	if err := history.Load(); err != nil {
		return history.GetStats(), &DownloadHistoryOperationError{Op: "load", Err: err}
	}
	for _, item := range items {
		e.processItem(item, history)
	}
	if err := history.Save(); err != nil {
		return history.GetStats(), &DownloadHistoryOperationError{Op: "save", Err: err}
	}
	return history.GetStats(), nil
}

// ExportRootsWithHistory is a wrapper for exporting multiple root directories with download history management.
// It converts rootIDs to ExportItem and delegates to exportItemsWithHistory.
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

// processDirectory recursively processes a directory and its subdirectories,
// exporting all convertible Synology Office files.
// Parameters:
//   - item: The ExportItem representing the directory to process
//   - history: DownloadHistory instance to record downloaded files
//
// Directory errors are logged and counted in history. Processing continues even if errors occur.
func (e *Exporter) processDirectory(item ExportItem, history *DownloadHistory) {
	list, err := e.session.List(item.FileID)
	if err != nil {
		fmt.Printf("Failed to list directory %s: %v\n", item.DisplayPath, err)
		history.ErrorCount.Increment()
		return
	}
	for _, child := range list.Items {
		e.processItem(toExportItem(child), history)
	}
}

// processFile exports a single convertible Synology Office file and updates the download history accordingly.
//
// Parameters:
//   - item: ExportItem representing the Synology Office file to export
//   - history: DownloadHistory instance to record and update download status
//
// If the file is not exportable, increments the ignored counter.
// If the file was already exported and hash is unchanged, increments the skipped counter and marks status as "skipped".
// Otherwise, exports and saves the file, then records or updates its status as "downloaded".
// All status transitions are performed via DownloadHistory methods with precondition checks.
func (e *Exporter) processFile(item ExportItem, history *DownloadHistory) {
	exportName := synd.GetExportFileName(item.DisplayPath)
	if exportName == "" {
		fmt.Printf("Skip (not exportable): %s\n", item.DisplayPath)
		history.IgnoredCount.Increment()
		return
	}

	localPath := strings.TrimPrefix(filepath.Clean(exportName), "/")
	if prev, downloaded := history.Items[localPath]; downloaded && prev.Hash == item.Hash {
		fmt.Printf("Skip (already exported and hash unchanged): %s\n", localPath)
		history.SkippedCount.Increment()
		// Mark as skipped only if current status is "loaded" (precondition checked in method)
		err := history.MarkSkipped(localPath)
		if err != nil {
			fmt.Printf("Warning: could not mark as skipped: %v\n", err)
		}
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
	newItem := DownloadItem{
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

// ExportItem contains only the fields needed for export processing.
// This reduces dependency on the full ResponseItem struct and improves maintainability.
type ExportItem struct {
	Type        synd.ObjectType
	FileID      synd.FileID
	DisplayPath string
	Hash        synd.FileHash
}

// toExportItem converts a *synd.ResponseItem to an ExportItem for export processing.
// Only the necessary fields are copied. This is a standalone function because Go does not allow methods on non-local types.
func toExportItem(item *synd.ResponseItem) ExportItem {
	return ExportItem{
		Type:        item.Type,
		FileID:      item.FileID,
		DisplayPath: item.DisplayPath,
		Hash:        item.Hash,
	}
}

// processItem processes a single item (file or directory).
// If the item is a directory, recursively processes its contents.
// If the item is an exportable file, exports and saves it.
// Errors are logged and processing continues.
//
// DownloadItem.Status is used to distinguish:
//   - "loaded": entry loaded from JSON but not touched in this export
//   - "downloaded": file was downloaded in this session
//   - "skipped": file was found and skipped (hash unchanged) in this session
func (e *Exporter) processItem(item ExportItem, history *DownloadHistory) {
	switch item.Type {
	case synd.ObjectTypeDirectory:
		e.processDirectory(item, history)
	case synd.ObjectTypeFile:
		e.processFile(item, history)
	}
}

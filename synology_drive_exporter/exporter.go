package synology_drive_exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/isseis/go-synology-office-exporter/download_history"
	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

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

// SessionInterface abstracts Synology session operations for export and testability.
type SessionInterface interface {
	// List retrieves items from the specified root directory.
	List(rootDirID synd.FileID) (*synd.ListResponse, error)

	// Export exports the specified file, performing format conversion if needed.
	Export(fileID synd.FileID) (*synd.ExportResponse, error)

	// TeamFolder retrieves team folders from the Synology Drive API.
	TeamFolder() (*synd.TeamFolderResponse, error)

	// SharedWithMe retrieves files shared with the user.
	SharedWithMe() (*synd.SharedWithMeResponse, error)
}

// Exporter handles exporting files from Synology Drive, maintaining download history and file system abstraction.
type Exporter struct {
	session     SessionInterface
	downloadDir string // Directory where downloaded files will be saved
	fs          FileSystemOperations
}

// NewExporter constructs an Exporter with a real Synology session and the specified download directory. If downloadDir is empty, the current directory is used.
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

// NewExporterWithDependencies constructs an Exporter with injected dependencies for session, download directory, and file system. Intended for testing and advanced use.
func NewExporterWithDependencies(session SessionInterface, downloadDir string, fs FileSystemOperations) *Exporter {
	return &Exporter{
		session:     session,
		downloadDir: downloadDir,
		fs:          fs,
	}
}

// ExportMyDrive exports convertible files from the user's Synology Drive, using download history to avoid duplicates.
func (e *Exporter) ExportMyDrive() (download_history.ExportStats, error) {
	return e.ExportRootsWithHistory(
		[]synd.FileID{synd.MyDrive},
		"mydrive_history.json",
	)
}

// ExportTeamFolder exports convertible files from all team folders, using download history to avoid duplicates.
func (e *Exporter) ExportTeamFolder() (download_history.ExportStats, error) {
	teamFolder, err := e.session.TeamFolder()
	if err != nil {
		return download_history.ExportStats{}, err
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

// ExportSharedWithMe exports convertible files and directories shared with the user, using download history to avoid duplicates.
func (e *Exporter) ExportSharedWithMe() (download_history.ExportStats, error) {
	sharedWithMe, err := e.session.SharedWithMe()
	if err != nil {
		return download_history.ExportStats{}, err
	}
	var exportItems []ExportItem
	for _, item := range sharedWithMe.Items {
		exportItems = append(exportItems, toExportItem(item))
	}
	return e.exportItemsWithHistory(exportItems, "shared_with_me_history.json")
}

// exportItemsWithHistory is an internal helper for exporting a slice of ExportItem with download history management.
func (e *Exporter) exportItemsWithHistory(
	items []ExportItem,
	historyFile string,
) (download_history.ExportStats, error) {
	historyPath := filepath.Join(e.downloadDir, historyFile)
	history, err := download_history.NewDownloadHistory(historyPath)
	if err != nil {
		// TODO: Replace with correct error type if DownloadHistoryOperationError is not available
		return download_history.ExportStats{}, fmt.Errorf("download history operation (create) failed: %w", err)
	}
	if err := history.Load(); err != nil {
		// TODO: Replace with correct error type if DownloadHistoryOperationError is not available
		return history.GetStats(), fmt.Errorf("download history operation (load) failed: %w", err)
	}
	for _, item := range items {
		e.processItem(item, history)
	}
	if err := history.Save(); err != nil {
		// TODO: Replace with correct error type if DownloadHistoryOperationError is not available
		return history.GetStats(), fmt.Errorf("download history operation (save) failed: %w", err)
	}
	return history.GetStats(), nil
}

// ExportRootsWithHistory exports multiple root directories with download history management.
func (e *Exporter) ExportRootsWithHistory(
	rootIDs []synd.FileID,
	historyFile string,
) (download_history.ExportStats, error) {
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

// processFile exports a single convertible file and updates download history. Handles export, skip, and error logic.
func (e *Exporter) processFile(item ExportItem, history *download_history.DownloadHistory) {
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

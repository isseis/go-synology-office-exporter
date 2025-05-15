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
		return fmt.Errorf("failed to create directories for %s: %w", filename, err)
	}

	// Write data to the file
	if err := os.WriteFile(filename, data, filePerm); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filename, err)
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
func (e *Exporter) ExportMyDrive() error {
	return e.ExportRootsWithHistory(
		[]synd.FileID{synd.MyDrive},
		"mydrive_history.json",
	)
}

// ExportTeamFolder exports all convertible files from all team folders.
// Download history is used to avoid duplicate downloads.
func (e *Exporter) ExportTeamFolder() error {
	teamFolder, err := e.session.TeamFolder()
	if err != nil {
		return err
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
func (e *Exporter) ExportSharedWithMe() error {
	sharedWithMe, err := e.session.SharedWithMe()
	if err != nil {
		return err
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
func (e *Exporter) exportItemsWithHistory(
	items []ExportItem,
	historyFile string,
) error {
	historyPath := filepath.Join(e.downloadDir, historyFile)
	history, err := NewDownloadHistory(historyPath)
	if err != nil {
		return fmt.Errorf("failed to create download history: %w", err)
	}
	if err := history.Load(); err != nil {
		return fmt.Errorf("failed to load download history: %w", err)
	}
	for _, item := range items {
		if err := e.processItem(item, history, true); err != nil {
			return err
		}
	}
	if err := history.Save(); err != nil {
		return fmt.Errorf("failed to save download history: %w", err)
	}
	return nil
}

// ExportRootsWithHistory is a wrapper for exporting multiple root directories with download history management.
// It converts rootIDs to ExportItem and delegates to exportItemsWithHistory.
func (e *Exporter) ExportRootsWithHistory(
	rootIDs []synd.FileID,
	historyFile string,
) error {
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
// Returns:
//   - error: An error if the export operation failed
//
// processDirectory recursively processes a directory and its subdirectories.
// If topLevel is true, errors are returned; otherwise, errors are logged and skipped.
func (e *Exporter) processDirectory(item ExportItem, history *DownloadHistory, topLevel bool) error {
	list, err := e.session.List(item.FileID)
	if err != nil {
		return err
	}
	for _, child := range list.Items {
		if err := e.processItem(toExportItem(child), history, false); err != nil {
			return err
		}
	}
	return nil
}

// processFile exports a single convertible Synology Office file.
// Parameters:
//   - item: The Synology Office file to export (ExportItem type fields required)
//   - history: DownloadHistory instance to record downloaded files
//
// Returns:
//   - error: An error if the export operation failed
func (e *Exporter) processFile(item ExportItem, history *DownloadHistory) error {
	exportName := synd.GetExportFileName(item.DisplayPath)
	if exportName == "" {
		return nil
	}

	localPath := strings.TrimPrefix(filepath.Clean(exportName), "/")
	if history != nil {
		if prev, downloaded := history.Items[localPath]; downloaded && prev.Hash == item.Hash {
			fmt.Printf("Skip (already exported and hash unchanged): %s\n", localPath)
			return nil
		}
	}
	fmt.Printf("Exporting file: %s\n", exportName)
	resp, err := e.session.Export(item.FileID)
	if err != nil {
		return fmt.Errorf("failed to export %s: %w", exportName, err)
	}
	downloadPath := filepath.Join(e.downloadDir, localPath)
	if err := e.fs.CreateFile(downloadPath, resp.Content, 0755, 0644); err != nil {
		return ExportFileWriteError(err.Error())
	}
	fmt.Printf("Saved to: %s\n", downloadPath)

	if history != nil {
		history.Items[localPath] = DownloadItem{
			FileID:       item.FileID,
			Hash:         item.Hash,
			DownloadTime: time.Now(),
		}
	}
	return nil
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
// Returns an error only if a file write fails.
// If topLevel is true, directory errors are returned; otherwise, errors are logged and processing continues.
func (e *Exporter) processItem(item ExportItem, history *DownloadHistory, topLevel bool) error {
	switch item.Type {
	case synd.ObjectTypeDirectory:
		if err := e.processDirectory(item, history, topLevel); err != nil {
			if topLevel {
				// Return error for top-level directory
				return err
			} else {
				fmt.Printf("Failed to process directory %s: %v\n", item.DisplayPath, err)
				// Continue processing other items even if one directory fails
			}
		}
	case synd.ObjectTypeFile:
		if err := e.processFile(item, history); err != nil {
			fmt.Printf("Failed to process file %s: %v\n", item.DisplayPath, err)
			// Continue processing other items even if one file fails
		}
	}
	return nil
}

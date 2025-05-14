package synology_drive_exporter

import (
	"fmt"
	"os"
	"path/filepath"

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
		return nil, err
	}
	if err = session.Login(); err != nil {
		return nil, err
	}

	exporter := &Exporter{
		session:     session,
		downloadDir: downloadDir,
		fs:          &DefaultFileSystem{},
	}
	return exporter, nil
}

// NewExporterWithCustomDependencies creates a new Exporter with custom dependencies for testing.
func NewExporterWithCustomDependencies(session SessionInterface, downloadDir string, fs FileSystemOperations) *Exporter {
	return &Exporter{
		session:     session,
		downloadDir: downloadDir,
		fs:          fs,
	}
}

// ExportMyDrive exports all convertible files from the user's Synology Drive
// and saves them to the download directory.
func (e *Exporter) ExportMyDrive() error {
	return e.processDirectory(synd.MyDrive)
}

// ExportTeamFolder exports all convertible files from the team folder.
func (e *Exporter) ExportTeamFolder() error {
	teamFolder, err := e.session.TeamFolder()
	if err != nil {
		return err
	}

	for _, item := range teamFolder.Items {
		if err := e.processDirectory(item.FileID); err != nil {
			return err
		}
	}
	return nil
}

// ExportSharedWithMe exports all convertible files and directories shared with the user.
// It processes both files and directories in the shared-with-me list.
func (e *Exporter) ExportSharedWithMe() error {
	sharedWithMe, err := e.session.SharedWithMe()
	if err != nil {
		return err
	}

	for _, item := range sharedWithMe.Items {
		if err := e.processItem(item); err != nil {
			return err
		}
	}
	return nil
}

// processDirectory recursively processes a directory and its subdirectories,
// exporting all convertible Synology Office files.
// Parameters:
//   - dirID: The identifier of the directory to process
//
// Returns:
//   - error: An error if the export operation failed
func (e *Exporter) processDirectory(dirID synd.FileID) error {
	list, err := e.session.List(dirID)
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if err := e.processItem(item); err != nil {
			return err
		}
	}
	return nil
}

// processItem processes a single item (file or directory).
// If the item is a directory, recursively processes its contents.
// If the item is an exportable file, exports and saves it.
// Returns an error only if a file write fails.
func (e *Exporter) processItem(item *synd.ResponseItem) error {
	// Use a tagged switch for item.Type for clarity and maintainability.
	switch item.Type {
	case synd.ObjectTypeDirectory:
		// Recursively process directory
		if err := e.processDirectory(item.FileID); err != nil {
			fmt.Printf("Failed to process directory %s: %v\n", item.DisplayPath, err)
			// Continue processing other items even if one directory fails
		}
	case synd.ObjectTypeFile:
		// Export file if convertible
		exportName := synd.GetExportFileName(item.DisplayPath)
		if exportName == "" {
			return nil
		}
		fmt.Printf("Exporting file: %s\n", exportName)
		resp, err := e.session.Export(item.FileID)
		if err != nil {
			fmt.Printf("Failed to export %s: %v\n", exportName, err)
			return nil
		}
		localPath := filepath.Clean(exportName)
		for len(localPath) > 0 && localPath[0] == '/' {
			localPath = localPath[1:]
		}
		downloadPath := filepath.Join(e.downloadDir, localPath)
		if err := e.fs.CreateFile(downloadPath, resp.Content, 0755, 0644); err != nil {
			return ExportFileWriteError(err.Error())
		}
		fmt.Printf("Saved to: %s\n", downloadPath)
	}
	return nil
}

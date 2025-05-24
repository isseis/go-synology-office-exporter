package synology_drive_exporter

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileSystemOperations abstracts file system operations for the Exporter, allowing for easy mocking in tests.
// FileSystemOperations abstracts file system operations for the Exporter, allowing for easy mocking in tests.
type FileSystemOperations interface {
	// CreateFile writes data to a file, creating parent directories if needed. Directory and file permissions are set by dirPerm and filePerm.
	CreateFile(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error
	// Remove deletes the specified file from the filesystem.
	Remove(path string) error
}

// DefaultFileSystem provides a production implementation of FileSystemOperations using the os package.
type DefaultFileSystem struct{}

// CreateFile writes data to a file, creating parent directories if needed.
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

// Remove deletes the specified file from the filesystem.
func (fs *DefaultFileSystem) Remove(path string) error {
	return os.Remove(path)
}

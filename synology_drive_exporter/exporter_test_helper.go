//go:build test
// +build test

package synology_drive_exporter

import (
	"errors"
	"os"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// MockFileSystem is a mock implementation of FileSystemOperations for testing.
// MockFileSystem is a mock implementation of FileSystemOperations for testing.
type MockFileSystem struct {
	CreateFileFunc func(string, []byte, os.FileMode, os.FileMode) error
	RemoveFunc     func(path string) error
	WrittenFiles   map[string][]byte
	RemovedFiles   map[string]bool
}

// NewMockFileSystem creates a new MockFileSystem with default no-op implementations.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		CreateFileFunc: func(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error {
			return nil
		},
		RemoveFunc: func(path string) error {
			return nil
		},
		WrittenFiles: make(map[string][]byte),
		RemovedFiles: make(map[string]bool),
	}
}

// CreateFile simulates file creation for testing. It records written files in WrittenFiles.
// CreateFile simulates file creation for testing. It records written files in WrittenFiles.
func (m *MockFileSystem) CreateFile(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error {
	if m.CreateFileFunc != nil {
		err := m.CreateFileFunc(filename, data, dirPerm, filePerm)
		if err == nil {
			m.WrittenFiles[filename] = data
		}
		return err
	}

	// If no custom function is provided, simulate file writing.
	m.WrittenFiles[filename] = data
	return nil
}

// Remove simulates file removal for testing.
func (m *MockFileSystem) Remove(path string) error {
	if m.RemoveFunc != nil {
		err := m.RemoveFunc(path)
		if err == nil {
			m.RemovedFiles[path] = true
		}
		return err
	}
	// If no custom function is provided, simulate file removal.
	m.RemovedFiles[path] = true
	return nil
}

type MockSynologySession struct {
	ListFunc         func(rootDirID synd.FileID, offset, limit int64) (*synd.ListResponse, error)
	ExportFunc       func(fileID synd.FileID) (*synd.ExportResponse, error)
	TeamFolderFunc   func(offset, limit int64) (*synd.TeamFolderResponse, error)
	SharedWithMeFunc func(offset, limit int64) (*synd.SharedWithMeResponse, error)
	MaxPageSize      int64
}

func (m *MockSynologySession) List(rootDirID synd.FileID, offset, limit int64) (*synd.ListResponse, error) {
	if m.ListFunc == nil {
		return nil, errors.New("ListFunc not set")
	}
	return m.ListFunc(rootDirID, offset, limit)
}

func (m *MockSynologySession) Export(fileID synd.FileID) (*synd.ExportResponse, error) {
	if m.ExportFunc != nil {
		return m.ExportFunc(fileID)
	}
	return nil, errors.New("ExportFunc not set")
}

func (m *MockSynologySession) TeamFolder(offset, limit int64) (*synd.TeamFolderResponse, error) {
	if m.TeamFolderFunc != nil {
		return m.TeamFolderFunc(offset, limit)
	}
	return nil, errors.New("TeamFolderFunc not set")
}

func (m *MockSynologySession) SharedWithMe(offset, limit int64) (*synd.SharedWithMeResponse, error) {
	if m.SharedWithMeFunc != nil {
		return m.SharedWithMeFunc(offset, limit)
	}
	return nil, errors.New("SharedWithMeFunc not set")
}

// GetMaxPageSize returns the maximum number of items that can be requested per page.
func (m *MockSynologySession) GetMaxPageSize() int64 {
	if m.MaxPageSize <= 0 {
		return synd.DefaultMaxPageSize
	}
	return m.MaxPageSize
}

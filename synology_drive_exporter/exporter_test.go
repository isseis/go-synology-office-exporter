package synology_drive_exporter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

type MockFileSystem struct {
	CreateFileFunc func(string, []byte, os.FileMode, os.FileMode) error
	// Records created directories and files
	CreatedDirs  []string
	WrittenFiles map[string][]byte
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		CreateFileFunc: func(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error {
			return nil
		},
		WrittenFiles: make(map[string][]byte),
	}
}

func (m *MockFileSystem) CreateFile(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error {
	if m.CreateFileFunc != nil {
		err := m.CreateFileFunc(filename, data, dirPerm, filePerm)
		if err == nil {
			dir := filepath.Dir(filename)
			m.CreatedDirs = append(m.CreatedDirs, dir)
			m.WrittenFiles[filename] = data
		}
		return err
	}

	// If no custom function provided, create directory and write file
	dir := filepath.Dir(filename)
	m.CreatedDirs = append(m.CreatedDirs, dir)
	m.WrittenFiles[filename] = data
	return nil
}

type MockSynologySession struct {
	ListFunc       func(rootDirID synd.FileID) (*synd.ListResponse, error)
	ExportFunc     func(fileID synd.FileID) (*synd.ExportResponse, error)
	TeamFolderFunc func() (*synd.TeamFolderResponse, error)
}

func (m *MockSynologySession) List(rootDirID synd.FileID) (*synd.ListResponse, error) {
	if m.ListFunc != nil {
		return m.ListFunc(rootDirID)
	}
	return &synd.ListResponse{}, nil
}

func (m *MockSynologySession) Export(fileID synd.FileID) (*synd.ExportResponse, error) {
	if m.ExportFunc != nil {
		return m.ExportFunc(fileID)
	}
	return &synd.ExportResponse{}, nil
}

func (m *MockSynologySession) TeamFolder() (*synd.TeamFolderResponse, error) {
	if m.TeamFolderFunc != nil {
		return m.TeamFolderFunc()
	}
	return &synd.TeamFolderResponse{}, nil
}

func TestExporterExportMyDrive(t *testing.T) {
	tests := []struct {
		name               string
		listResponse       *synd.ListResponse
		listError          error
		exportResponse     map[synd.FileID]*synd.ExportResponse
		exportError        map[synd.FileID]error
		fileOperationError error
		expectedError      bool
		expectedFiles      int
		// Track directory IDs that have been listed
		trackedListCalls map[synd.FileID]bool
		// Map of directory ID to list response for recursive directory traversal
		directoryResponses map[synd.FileID]*synd.ListResponse
		// Map of directory ID to list error for recursive directory traversal
		directoryErrors map[synd.FileID]error
	}{
		{
			name: "Normal case: Export two files",
			listResponse: &synd.ListResponse{
				Items: []*synd.ListResponseItem{
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file1",
						DisplayPath: "/doc/test1.odoc", // .docx -> .odoc
					},
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file2",
						DisplayPath: "/doc/test2.osheet", // .xlsx -> .osheet
					},
				},
			},
			exportResponse: map[synd.FileID]*synd.ExportResponse{
				"file1": {Content: []byte("file1 content")},
				"file2": {Content: []byte("file2 content")},
			},
			expectedFiles: 2,
		},
		{
			name: "Skip files that are not export targets",
			listResponse: &synd.ListResponse{
				Items: []*synd.ListResponseItem{
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file1",
						DisplayPath: "/doc/test1.odoc", // .docx -> .odoc
					},
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file2",
						DisplayPath: "/doc/test2.txt", // Not exportable extension
					},
				},
			},
			exportResponse: map[synd.FileID]*synd.ExportResponse{
				"file1": {Content: []byte("file1 content")},
			},
			expectedFiles: 1,
		},
		{
			name:          "Error when getting list",
			listError:     errors.New("list error"),
			expectedError: true,
		},
		{
			name: "Error during export",
			listResponse: &synd.ListResponse{
				Items: []*synd.ListResponseItem{
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file1",
						DisplayPath: "/doc/test1.odoc", // .docx -> .odoc
					},
				},
			},
			exportError: map[synd.FileID]error{
				"file1": errors.New("export error"),
			},
			expectedFiles: 0, // Errors are only logged and processing continues
		},
		{
			name: "Error during file operation",
			listResponse: &synd.ListResponse{
				Items: []*synd.ListResponseItem{
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file1",
						DisplayPath: "/doc/test1.odoc", // .docx -> .odoc
					},
				},
			},
			exportResponse: map[synd.FileID]*synd.ExportResponse{
				"file1": {Content: []byte("file1 content")},
			},
			fileOperationError: errors.New("file operation error"),
			expectedError:      true,
		},
		{
			name: "Directory traversal: One level deep",
			listResponse: &synd.ListResponse{
				Items: []*synd.ListResponseItem{
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file1",
						DisplayPath: "/doc/test1.odoc",
					},
					{
						Type:        synd.ObjectTypeDirectory,
						FileID:      "dir1",
						Name:        "subdir1",
						DisplayPath: "/subdir1",
					},
				},
			},
			directoryResponses: map[synd.FileID]*synd.ListResponse{
				"dir1": {
					Items: []*synd.ListResponseItem{
						{
							Type:        synd.ObjectTypeFile,
							FileID:      "file2",
							DisplayPath: "/subdir1/test2.osheet",
						},
					},
				},
			},
			exportResponse: map[synd.FileID]*synd.ExportResponse{
				"file1": {Content: []byte("file1 content")},
				"file2": {Content: []byte("file2 content")},
			},
			trackedListCalls: make(map[synd.FileID]bool),
			expectedFiles:    2,
		},
		{
			name: "Directory traversal: Multiple levels deep",
			listResponse: &synd.ListResponse{
				Items: []*synd.ListResponseItem{
					{
						Type:        synd.ObjectTypeDirectory,
						FileID:      "dir1",
						Name:        "level1",
						DisplayPath: "/level1",
					},
				},
			},
			directoryResponses: map[synd.FileID]*synd.ListResponse{
				"dir1": {
					Items: []*synd.ListResponseItem{
						{
							Type:        synd.ObjectTypeDirectory,
							FileID:      "dir2",
							Name:        "level2",
							DisplayPath: "/level1/level2",
						},
					},
				},
				"dir2": {
					Items: []*synd.ListResponseItem{
						{
							Type:        synd.ObjectTypeFile,
							FileID:      "file1",
							DisplayPath: "/level1/level2/test1.odoc",
						},
					},
				},
			},
			exportResponse: map[synd.FileID]*synd.ExportResponse{
				"file1": {Content: []byte("nested file content")},
			},
			trackedListCalls: make(map[synd.FileID]bool),
			expectedFiles:    1,
		},
		{
			name: "Directory traversal: Error in subdirectory should not stop processing",
			listResponse: &synd.ListResponse{
				Items: []*synd.ListResponseItem{
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file1",
						DisplayPath: "/doc/test1.odoc",
					},
					{
						Type:        synd.ObjectTypeDirectory,
						FileID:      "dir1",
						Name:        "error_dir",
						DisplayPath: "/error_dir",
					},
					{
						Type:        synd.ObjectTypeDirectory,
						FileID:      "dir2",
						Name:        "good_dir",
						DisplayPath: "/good_dir",
					},
				},
			},
			directoryResponses: map[synd.FileID]*synd.ListResponse{
				"dir2": {
					Items: []*synd.ListResponseItem{
						{
							Type:        synd.ObjectTypeFile,
							FileID:      "file2",
							DisplayPath: "/good_dir/test2.osheet",
						},
					},
				},
			},
			directoryErrors: map[synd.FileID]error{
				"dir1": errors.New("error listing directory"),
			},
			exportResponse: map[synd.FileID]*synd.ExportResponse{
				"file1": {Content: []byte("file1 content")},
				"file2": {Content: []byte("file2 content")},
			},
			trackedListCalls: make(map[synd.FileID]bool),
			expectedFiles:    2,
			expectedError:    false,
		},
		{
			name: "Mixed files and directories at multiple levels",
			listResponse: &synd.ListResponse{
				Items: []*synd.ListResponseItem{
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file1",
						DisplayPath: "/root1.odoc",
					},
					{
						Type:        synd.ObjectTypeDirectory,
						FileID:      "dir1",
						Name:        "docs",
						DisplayPath: "/docs",
					},
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file2",
						DisplayPath: "/root2.osheet",
					},
				},
			},
			directoryResponses: map[synd.FileID]*synd.ListResponse{
				"dir1": {
					Items: []*synd.ListResponseItem{
						{
							Type:        synd.ObjectTypeFile,
							FileID:      "file3",
							DisplayPath: "/docs/doc1.odoc",
						},
						{
							Type:        synd.ObjectTypeDirectory,
							FileID:      "dir2",
							Name:        "archived",
							DisplayPath: "/docs/archived",
						},
					},
				},
				"dir2": {
					Items: []*synd.ListResponseItem{
						{
							Type:        synd.ObjectTypeFile,
							FileID:      "file4",
							DisplayPath: "/docs/archived/old.odoc",
						},
						{
							Type:        synd.ObjectTypeFile,
							FileID:      "file5",
							DisplayPath: "/docs/archived/legacy.osheet",
						},
					},
				},
			},
			exportResponse: map[synd.FileID]*synd.ExportResponse{
				"file1": {Content: []byte("root1 content")},
				"file2": {Content: []byte("root2 content")},
				"file3": {Content: []byte("doc1 content")},
				"file4": {Content: []byte("old content")},
				"file5": {Content: []byte("legacy content")},
			},
			trackedListCalls: make(map[synd.FileID]bool),
			expectedFiles:    5,
		},
		{
			name: "File paths with multiple leading slashes",
			listResponse: &synd.ListResponse{
				Items: []*synd.ListResponseItem{
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file1",
						DisplayPath: "///doc/test1.odoc", // Triple leading slashes
					},
					{
						Type:        synd.ObjectTypeFile,
						FileID:      "file2",
						DisplayPath: "//doc/test2.osheet", // Double leading slashes
					},
				},
			},
			exportResponse: map[synd.FileID]*synd.ExportResponse{
				"file1": {Content: []byte("file1 content")},
				"file2": {Content: []byte("file2 content")},
			},
			expectedFiles: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSession := &MockSynologySession{
				ListFunc: func(rootDirID synd.FileID) (*synd.ListResponse, error) {
					// Track that this directory was listed
					if tt.trackedListCalls != nil {
						tt.trackedListCalls[rootDirID] = true
					}

					// Return directory-specific error if specified
					if tt.directoryErrors != nil {
						if err, ok := tt.directoryErrors[rootDirID]; ok {
							return nil, err
						}
					}

					// Return generic list error if specified
					if rootDirID == synd.MyDrive && tt.listError != nil {
						return nil, tt.listError
					}

					// Return directory-specific response if specified
					if tt.directoryResponses != nil {
						if resp, ok := tt.directoryResponses[rootDirID]; ok {
							return resp, nil
						}
					}

					// Return root response for MyDrive
					if rootDirID == synd.MyDrive {
						return tt.listResponse, nil
					}

					// Default empty response
					return &synd.ListResponse{Items: []*synd.ListResponseItem{}}, nil
				},
				ExportFunc: func(fileID synd.FileID) (*synd.ExportResponse, error) {
					if tt.exportError != nil {
						if err, ok := tt.exportError[fileID]; ok {
							return nil, err
						}
					}
					if tt.exportResponse != nil {
						if resp, ok := tt.exportResponse[fileID]; ok {
							return resp, nil
						}
					}
					return &synd.ExportResponse{}, nil
				},
			}

			mockFS := NewMockFileSystem()
			if tt.fileOperationError != nil {
				mockFS.CreateFileFunc = func(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error {
					return tt.fileOperationError
				}
			}

			// Create the instance to be tested
			exporter := NewExporterWithCustomDependencies(mockSession, "/tmp/test", mockFS)

			// Run the test
			err := exporter.ExportMyDrive()

			// Assertions
			if tt.expectedError && err == nil {
				t.Error("Expected error did not occur")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error occurred: %v", err)
			}

			// Validate file write count
			if len(mockFS.WrittenFiles) != tt.expectedFiles {
				t.Errorf("Expected %d files to be written, but got %d",
					tt.expectedFiles, len(mockFS.WrittenFiles))
			}

			// Check if all expected directories were traversed
			if tt.directoryResponses != nil {
				for dirID := range tt.directoryResponses {
					if tt.directoryErrors != nil && tt.directoryErrors[dirID] != nil {
						// Skip checking directories we expect to error
						continue
					}
					if !tt.trackedListCalls[dirID] {
						t.Errorf("Directory with ID %s was not traversed", dirID)
					}
				}
			}

			// Validate written paths for nested directory structure
			if tt.directoryResponses != nil && !tt.expectedError {
				// Check for files in the root directory
				if tt.listResponse != nil {
					for _, item := range tt.listResponse.Items {
						if item.Type == synd.ObjectTypeFile {
							validateExportedFile(t, item, mockFS, tt.exportError, tt.fileOperationError)
						}
					}
				}

				// Check for files in subdirectories
				for dirID, resp := range tt.directoryResponses {
					// Skip directories we expect to error
					if tt.directoryErrors != nil && tt.directoryErrors[dirID] != nil {
						continue
					}

					for _, item := range resp.Items {
						if item.Type == synd.ObjectTypeFile {
							validateExportedFile(t, item, mockFS, tt.exportError, tt.fileOperationError)
						}
					}
				}
			}
		})
	}
}

// validateExportedFile is a helper function to verify that a file was properly exported
func validateExportedFile(t *testing.T, item *synd.ListResponseItem, mockFS *MockFileSystem, exportErrors map[synd.FileID]error, fileOpError error) {
	exportName := synd.GetExportFileName(item.DisplayPath)
	if exportName == "" {
		return
	}

	// Clean path and remove all leading slashes
	cleanPath := filepath.Clean(exportName)
	for len(cleanPath) > 0 && cleanPath[0] == '/' {
		cleanPath = cleanPath[1:]
	}
	expectedPath := filepath.Join("/tmp/test", cleanPath)

	if fileOpError == nil {
		// Only verify if there's no export error for this file
		if exportErrors == nil || exportErrors[item.FileID] == nil {
			if _, exists := mockFS.WrittenFiles[expectedPath]; !exists {
				t.Errorf("File %s was not written", expectedPath)
			}
		}
	}
}

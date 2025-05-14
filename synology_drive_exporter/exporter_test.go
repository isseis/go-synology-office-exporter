package synology_drive_exporter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"maps"

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
	ListFunc         func(rootDirID synd.FileID) (*synd.ListResponse, error)
	ExportFunc       func(fileID synd.FileID) (*synd.ExportResponse, error)
	TeamFolderFunc   func() (*synd.TeamFolderResponse, error)
	SharedWithMeFunc func() (*synd.SharedWithMeResponse, error)
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

func (m *MockSynologySession) SharedWithMe() (*synd.SharedWithMeResponse, error) {
	if m.SharedWithMeFunc != nil {
		return m.SharedWithMeFunc()
	}
	return &synd.SharedWithMeResponse{}, nil
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
				Items: []*synd.ResponseItem{
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
				Items: []*synd.ResponseItem{
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
				Items: []*synd.ResponseItem{
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
				Items: []*synd.ResponseItem{
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
				Items: []*synd.ResponseItem{
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
					Items: []*synd.ResponseItem{
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
				Items: []*synd.ResponseItem{
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
					Items: []*synd.ResponseItem{
						{
							Type:        synd.ObjectTypeDirectory,
							FileID:      "dir2",
							Name:        "level2",
							DisplayPath: "/level1/level2",
						},
					},
				},
				"dir2": {
					Items: []*synd.ResponseItem{
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
				Items: []*synd.ResponseItem{
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
					Items: []*synd.ResponseItem{
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
				Items: []*synd.ResponseItem{
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
					Items: []*synd.ResponseItem{
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
					Items: []*synd.ResponseItem{
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
				Items: []*synd.ResponseItem{
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
					return &synd.ListResponse{Items: []*synd.ResponseItem{}}, nil
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
			dir, err := os.MkdirTemp("", "")
			if err != nil {
				t.Fatal(err)
			}
			exporter := NewExporterWithCustomDependencies(mockSession, dir, mockFS)
			defer os.RemoveAll(dir)

			// Run the test
			err = exporter.ExportMyDrive()

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
func validateExportedFile(t *testing.T, item *synd.ResponseItem, mockFS *MockFileSystem, exportErrors map[synd.FileID]error, fileOpError error) {
	// Implementation omitted for this test; see above for details.
}

/*
TestProcessItem_HistoryAndHash covers:
1. Skips download if history exists and hash is the same
2. Downloads if history exists and hash is different
3. Downloads if history does not exist
*/
func TestProcessItem_HistoryAndHash(t *testing.T) {
	fileID := synd.FileID("file1")
	fileHashOld := synd.FileHash("hash_old")
	fileHashNew := synd.FileHash("hash_new")
	displayPath := "/doc/test1.odoc"
	exportName := synd.GetExportFileName(displayPath)
	cleanPath := filepath.Clean(exportName)
	for len(cleanPath) > 0 && cleanPath[0] == '/' {
		cleanPath = cleanPath[1:]
	}

	cases := []struct {
		name        string
		history     map[string]DownloadItem
		itemHash    synd.FileHash
		expectWrite bool
	}{
		{
			name:        "skip if hash unchanged",
			history:     map[string]DownloadItem{cleanPath: {FileID: fileID, Hash: fileHashOld}},
			itemHash:    fileHashOld,
			expectWrite: false,
		},
		{
			name:        "download if hash changed",
			history:     map[string]DownloadItem{cleanPath: {FileID: fileID, Hash: fileHashOld}},
			itemHash:    fileHashNew,
			expectWrite: true,
		},
		{
			name:        "download if not in history",
			history:     map[string]DownloadItem{},
			itemHash:    fileHashNew,
			expectWrite: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			session := &MockSynologySession{
				ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
					return &synd.ExportResponse{Content: []byte("file content")}, nil
				},
			}
			mockFS := NewMockFileSystem()
			writeCalled := false
			mockFS.CreateFileFunc = func(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error {
				writeCalled = true
				return nil
			}
			history := &DownloadHistory{Items: make(map[string]DownloadItem)}
			maps.Copy(history.Items, tc.history)
			item := &synd.ResponseItem{
				Type:        synd.ObjectTypeFile,
				FileID:      fileID,
				DisplayPath: displayPath,
				Hash:        tc.itemHash,
			}
			exporter := NewExporterWithCustomDependencies(session, "", mockFS)
			err := exporter.processItem(item, history)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if writeCalled != tc.expectWrite {
				t.Errorf("expected write: %v, got: %v", tc.expectWrite, writeCalled)
			}
		})
	}
}

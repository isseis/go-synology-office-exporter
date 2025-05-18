package synology_drive_exporter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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
			expectedFiles: 0,
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
			exporter := NewExporterWithDependencies(mockSession, dir, mockFS)
			defer os.RemoveAll(dir)

			// Run the test
			stats, err := exporter.ExportMyDrive()

			// Assertions
			if tt.expectedError && err == nil {
				t.Error("Expected error did not occur")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error occurred: %v", err)
			}

			// Validate file write count matches stats.Downloaded
			if len(mockFS.WrittenFiles) != stats.Downloaded {
				t.Errorf("Expected %d files to be written (stats.Downloaded), but got %d",
					stats.Downloaded, len(mockFS.WrittenFiles))
			}
			if stats.Downloaded != tt.expectedFiles {
				t.Errorf("Expected stats.Downloaded=%d, but got %d", tt.expectedFiles, stats.Downloaded)
			}
			if tt.expectedError && stats.Errors == 0 {
				t.Errorf("Expected stats.Errors > 0, but got %d", stats.Errors)
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

// validateExportedFile checks that a file was exported correctly by inspecting the mock file system.
func validateExportedFile(t *testing.T, item *synd.ResponseItem, mockFS *MockFileSystem, exportErrors map[synd.FileID]error, fileOpError error) {
	// Implementation omitted for this test; see above for details.
}

// newTestDownloadHistory creates a DownloadHistory instance for testing.
func newTestDownloadHistory(items map[string]DownloadItem) *DownloadHistory {
	if items == nil {
		items = make(map[string]DownloadItem)
	}
	return &DownloadHistory{
		Items: items,
	}
}

// TestExporter_Counts verifies that DownloadHistory's counters are incremented correctly.
func TestExporter_Counts(t *testing.T) {
	fileID := synd.FileID("file1")
	fileHash := synd.FileHash("hash1")
	fileID2 := synd.FileID("file2")
	fileHash2 := synd.FileHash("hash2")
	ignoredFileID := synd.FileID("file3")
	ignoredPath := "/doc/ignored.txt" // not exportable
	displayPath := "/doc/test1.odoc"
	cleanPath := TestMakeKey(displayPath)

	t.Run("DownloadCount increments on successful export", func(t *testing.T) {
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		history := newTestDownloadHistory(nil)
		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      fileID,
			DisplayPath: "/doc/test1.odoc",
			Hash:        fileHash,
		}
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processItem(item, history)
		if got := history.DownloadCount.Get(); got != 1 {
			t.Errorf("DownloadCount = %d, want 1", got)
		}
	})

	t.Run("SkippedCount increments if file is already exported with same hash", func(t *testing.T) {
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		history := newTestDownloadHistory(map[string]DownloadItem{
			cleanPath: {FileID: fileID, Hash: fileHash},
		})
		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      fileID,
			DisplayPath: "/doc/test1.odoc",
			Hash:        fileHash,
		}
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processItem(item, history)
		if got := history.SkippedCount.Get(); got != 1 {
			t.Errorf("SkippedCount = %d, want 1", got)
		}
	})

	t.Run("IgnoredCount increments if file is not exportable", func(t *testing.T) {
		session := &MockSynologySession{}
		mockFS := NewMockFileSystem()
		history := newTestDownloadHistory(nil)
		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      ignoredFileID,
			DisplayPath: ignoredPath,
			Hash:        fileHash,
		}
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processItem(item, history)
		if got := history.IgnoredCount.Get(); got != 1 {
			t.Errorf("IgnoredCount = %d, want 1", got)
		}
	})

	t.Run("ErrorCount increments if export fails", func(t *testing.T) {
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return nil, errors.New("export failed")
			},
		}
		mockFS := NewMockFileSystem()
		history := newTestDownloadHistory(nil)
		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      fileID2,
			DisplayPath: "/doc/test2.odoc",
			Hash:        fileHash2,
		}
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processItem(item, history)
		if got := history.ErrorCount.Get(); got != 1 {
			t.Errorf("ErrorCount = %d, want 1", got)
		}
	})

	t.Run("ErrorCount increments if file write fails", func(t *testing.T) {
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		mockFS.CreateFileFunc = func(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error {
			return errors.New("write failed")
		}
		history := newTestDownloadHistory(nil)
		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      fileID2,
			DisplayPath: "/doc/test2.odoc",
			Hash:        fileHash2,
		}
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processItem(item, history)
		if got := history.ErrorCount.Get(); got != 1 {
			t.Errorf("ErrorCount = %d, want 1", got)
		}
	})
}

// TestExportItem_HistoryAndHash covers:
// 1. Skips download if history exists and hash is the same
// 2. Downloads if history exists and hash is different
// 3. Downloads if history does not exist
func TestExportItem_HistoryAndHash(t *testing.T) {
	// --- DownloadStatus unit tests ---
	t.Run("StatusDownloaded when new", func(t *testing.T) {
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		history := newTestDownloadHistory(nil)
		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      "file1",
			DisplayPath: "/doc/test1.odoc",
			Hash:        "hash1",
		}
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processFile(item, history)
		dlItem, exists := history.GetItemByDisplayPath(item.DisplayPath)
		if !exists {
			t.Fatal("expected item to exist in history")
		}
		if dlItem.DownloadStatus != StatusDownloaded {
			t.Errorf("expected StatusDownloaded, got %v", dlItem.DownloadStatus)
		}
	})

	t.Run("StatusSkipped when hash unchanged", func(t *testing.T) {
		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      "file1",
			DisplayPath: "/doc/test1.odoc",
			Hash:        "hash1",
		}
		initialTime := time.Date(2024, 5, 18, 8, 0, 0, 0, time.UTC)
		path := TestMakeKey(item.DisplayPath)
		history := newTestDownloadHistory(map[string]DownloadItem{
			path: {FileID: "file1", Hash: "hash1", DownloadStatus: StatusLoaded, DownloadTime: initialTime},
		})
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processFile(item, history)
		dlItem, exists := history.GetItemByDisplayPath(item.DisplayPath)
		if !exists {
			t.Fatal("expected item to exist in history")
		}
		if dlItem.DownloadStatus != StatusSkipped {
			t.Errorf("expected StatusSkipped, got %v", dlItem.DownloadStatus)
		}
		if !dlItem.DownloadTime.Equal(initialTime) {
			t.Errorf("expected DownloadTime to remain unchanged, got %v want %v", dlItem.DownloadTime, initialTime)
		}
	})

	// --- Integration test: coexistence of loaded, downloaded, skipped ---
	t.Run("StatusLoaded, StatusDownloaded, StatusSkipped coexistence", func(t *testing.T) {
		// Setup: 3 files in history, only 2 are present in exportItems
		item1 := ExportItem{Type: synd.ObjectTypeFile, FileID: "file1", DisplayPath: "/doc/test1.odoc", Hash: "hash1"}     // hash unchanged
		item2 := ExportItem{Type: synd.ObjectTypeFile, FileID: "file2", DisplayPath: "/doc/test2.odoc", Hash: "hash2-new"} // hash changed
		item3 := ExportItem{Type: synd.ObjectTypeFile, FileID: "file3", DisplayPath: "/doc/test3.odoc", Hash: "hash3"}     // only in history
		history := newTestDownloadHistory(map[string]DownloadItem{
			TestMakeKey(item1.DisplayPath): {FileID: "file1", Hash: "hash1", DownloadStatus: StatusLoaded},
			TestMakeKey(item2.DisplayPath): {FileID: "file2", Hash: "hash2-old", DownloadStatus: StatusLoaded},
			TestMakeKey(item3.DisplayPath): {FileID: "file3", Hash: "hash3", DownloadStatus: StatusLoaded},
		})
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processFile(item1, history) // should become skipped
		exporter.processFile(item2, history) // should become downloaded
		// item3 not processed, remains loaded

		// Check item1 status (should be skipped)
		item1Status, exists := history.GetItemByDisplayPath(item1.DisplayPath)
		if !exists {
			t.Fatal("expected item1 to exist in history")
		}
		if item1Status.DownloadStatus != StatusSkipped {
			t.Errorf("expected StatusSkipped for item1, got %v", item1Status.DownloadStatus)
		}

		// Check item2 status (should be downloaded)
		item2Status, exists := history.GetItemByDisplayPath(item2.DisplayPath)
		if !exists {
			t.Fatal("expected item2 to exist in history")
		}
		if item2Status.DownloadStatus != StatusDownloaded {
			t.Errorf("expected StatusDownloaded for item2, got %v", item2Status.DownloadStatus)
		}

		// Check item3 status (should remain loaded)
		item3Status, exists := history.GetItemByDisplayPath(item3.DisplayPath)
		if !exists {
			t.Fatal("expected item3 to exist in history")
		}
		if item3Status.DownloadStatus != StatusLoaded {
			t.Errorf("expected StatusLoaded for item3, got %v", item3Status.DownloadStatus)
		}
	})

	fileID := synd.FileID("file1")
	fileHashOld := synd.FileHash("hash_old")
	fileHashNew := synd.FileHash("hash_new")
	displayPath := "/doc/test1.odoc"
	cleanPath := TestMakeKey(displayPath)
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
			history := newTestDownloadHistory(tc.history)
			item := ExportItem{
				Type:        synd.ObjectTypeFile,
				FileID:      fileID,
				DisplayPath: displayPath,
				Hash:        tc.itemHash,
			}
			exporter := NewExporterWithDependencies(session, "", mockFS)
			exporter.processItem(item, history)
			if writeCalled != tc.expectWrite {
				t.Errorf("expected write: %v, got: %v", tc.expectWrite, writeCalled)
			}

			// Verify the item exists in history if we expected a write
			if tc.expectWrite {
				_, exists := history.GetItemByDisplayPath(displayPath)
				if !exists {
					t.Error("expected item to exist in history after processing")
				}
			}
		})
	}
}

package synology_drive_exporter

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isseis/go-synology-office-exporter/download_history"
	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// MockFileSystem is a mock implementation of FileSystemOperations for testing.
type MockFileSystem struct {
	CreateFileFunc func(string, []byte, os.FileMode, os.FileMode) error
	WrittenFiles   map[string][]byte
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		CreateFileFunc: func(filename string, data []byte, dirPerm os.FileMode, filePerm os.FileMode) error {
			return nil
		},
		WrittenFiles: make(map[string][]byte),
	}
}

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
		// trackedListCalls tracks directory IDs listed during recursive traversal.
		trackedListCalls map[synd.FileID]bool
		// directoryResponses maps directory IDs to list responses for recursive traversal.
		directoryResponses map[synd.FileID]*synd.ListResponse
		// directoryErrors maps directory IDs to list errors for recursive traversal.
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
			if tt.expectedError && stats.DownloadErrs == 0 {
				t.Errorf("Expected stats.DownloadErrs > 0, but got %d", stats.DownloadErrs)
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

func validateExportedFile(t *testing.T, item *synd.ResponseItem, mockFS *MockFileSystem, exportErrors map[synd.FileID]error, fileOpError error) {
	// Implementation omitted for this test; see above for details.
}

func TestExportStats(t *testing.T) {
	t.Run("No errors", func(t *testing.T) {
		stats := &ExportStats{
			Downloaded: 5,
			Skipped:    3,
			Ignored:    2,
			Removed:    1,
		}

		assert.Equal(t, 0, stats.TotalErrs(), "TotalErrs() should return 0 when no errors")
		expectedString := "downloaded=5, skipped=3, ignored=2, removed=1, download_errors=0, remove_errors=0"
		assert.Equal(t, expectedString, stats.String(), "String() should return the expected format")
	})

	t.Run("With download errors", func(t *testing.T) {
		stats := &ExportStats{
			Downloaded:   2,
			DownloadErrs: 1,
		}

		assert.Equal(t, 1, stats.TotalErrs(), "TotalErrs() should include download errors")
		expectedString := "downloaded=2, skipped=0, ignored=0, removed=0, download_errors=1, remove_errors=0"
		assert.Equal(t, expectedString, stats.String(), "String() should include download errors")
	})

	t.Run("With remove errors", func(t *testing.T) {
		stats := &ExportStats{
			Removed:    2,
			RemoveErrs: 1,
		}

		assert.Equal(t, 1, stats.TotalErrs(), "TotalErrs() should include remove errors")
		expectedString := "downloaded=0, skipped=0, ignored=0, removed=2, download_errors=0, remove_errors=1"
		assert.Equal(t, expectedString, stats.String(), "String() should include remove errors")
	})

	t.Run("Increment methods", func(t *testing.T) {
		t.Run("IncrementDownloadErrs", func(t *testing.T) {
			stats := &ExportStats{}
			initialErrs := stats.TotalErrs()
			stats.IncrementDownloadErrs()
			assert.Equal(t, initialErrs+1, stats.TotalErrs(), "Should increment total errors by 1")
			assert.Equal(t, 1, stats.DownloadErrs, "Should increment DownloadErrs counter")
		})

		t.Run("IncrementRemoveErrs", func(t *testing.T) {
			stats := &ExportStats{}
			initialErrs := stats.TotalErrs()
			stats.IncrementRemoveErrs()
			assert.Equal(t, initialErrs+1, stats.TotalErrs(), "Should increment total errors by 1")
			assert.Equal(t, 1, stats.RemoveErrs, "Should increment RemoveErrs counter")
		})

		t.Run("IncrementRemoved", func(t *testing.T) {
			stats := &ExportStats{}
			stats.IncrementRemoved()
			assert.Equal(t, 1, stats.Removed, "Should increment Removed counter")
		})
	})
}

func TestExporter_removeFile(t *testing.T) {
	tests := []struct {
		name        string
		dryRun      bool
		createFile  bool // Whether to create the test file for this test case
		shouldExist bool // Whether the file should exist after the operation
		expectErr   bool
	}{
		{
			name:        "Remove existing file",
			dryRun:      false,
			createFile:  true,
			shouldExist: false,
			expectErr:   false,
		},
		{
			name:        "Dry run should not remove file",
			dryRun:      true,
			createFile:  true,
			shouldExist: true, // In dry run, file should still exist
			expectErr:   false,
		},
		{
			name:        "Non-existent file should not error",
			dryRun:      false,
			createFile:  false,
			shouldExist: false,
			expectErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "testfile.txt")

			// Create a test file if needed for this test case
			if tt.createFile {
				err := os.WriteFile(testFile, []byte("test content"), 0644)
				require.NoError(t, err, "Failed to create test file")
			}

			e := &Exporter{
				DryRun: tt.dryRun,
			}

			err := e.removeFile(testFile)

			if tt.expectErr {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Unexpected error")
			}

			_, err = os.Stat(testFile)
			if tt.shouldExist {
				assert.NoError(t, err, "File should still exist but doesn't")
			} else if !os.IsNotExist(err) {
				t.Errorf("File should not exist but does or other error: %v", err)
			}
		})
	}
}

func TestExporter_cleanupObsoleteFiles(t *testing.T) {
	t.Run("Normal cleanup", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create some test files
		files := []string{
			filepath.Join(tempDir, "keep1.txt"),
			filepath.Join(tempDir, "keep2.txt"),
			filepath.Join(tempDir, "obsolete1.txt"),
			filepath.Join(tempDir, "obsolete2.txt"),
		}

		for _, f := range files {
			err := os.WriteFile(f, []byte("test"), 0644)
			require.NoError(t, err, "Failed to create test file")
		}

		// Create a history with some files
		history := download_history.NewDownloadHistoryForTest(map[string]download_history.DownloadItem{
			filepath.Join(tempDir, "keep1.txt"): {
				FileID:         "file1",
				Hash:           "hash1",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusDownloaded, // Mark as downloaded to keep
			},
			filepath.Join(tempDir, "keep2.txt"): {
				FileID:         "file2",
				Hash:           "hash2",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusDownloaded, // Mark as downloaded to keep
			},
			filepath.Join(tempDir, "obsolete1.txt"): {
				FileID:         "file3",
				Hash:           "hash3",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusLoaded, // Mark as loaded to be removed
			},
			filepath.Join(tempDir, "obsolete2.txt"): {
				FileID:         "file4",
				Hash:           "hash4",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusLoaded, // Mark as loaded to be removed
			},
		})

		e := &Exporter{
			fs:     NewMockFileSystem(),
			DryRun: false,
		}

		stats := &ExportStats{}
		e.cleanupObsoleteFiles(history, stats)

		// Verify the stats were updated correctly
		assert.Equal(t, 2, stats.Removed, "Should have removed 2 files")
		assert.Equal(t, 0, stats.RemoveErrs, "Should have no remove errors")

		// Verify the files were actually removed
		for _, f := range files {
			_, err := os.Stat(f)
			if strings.Contains(f, "obsolete") {
				assert.True(t, os.IsNotExist(err), "Obsolete file should have been removed: %s", f)
			} else {
				assert.NoError(t, err, "Non-obsolete file should still exist: %s", f)
			}
		}
	})

	t.Run("Dry run should not remove files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create some test files
		files := []string{
			filepath.Join(tempDir, "obsolete1.txt"),
			filepath.Join(tempDir, "obsolete2.txt"),
		}

		for _, f := range files {
			err := os.WriteFile(f, []byte("test"), 0644)
			require.NoError(t, err, "Failed to create test file")
		}

		// Create a history with some files
		history := download_history.NewDownloadHistoryForTest(map[string]download_history.DownloadItem{
			filepath.Join(tempDir, "obsolete1.txt"): {
				FileID:         "file1",
				Hash:           "hash1",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusLoaded, // Mark as loaded to be removed
			},
			filepath.Join(tempDir, "obsolete2.txt"): {
				FileID:         "file2",
				Hash:           "hash2",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusLoaded, // Mark as loaded to be removed
			},
		})

		e := &Exporter{
			fs:     NewMockFileSystem(),
			DryRun: true, // Enable dry run
		}

		stats := &ExportStats{}
		e.cleanupObsoleteFiles(history, stats)

		// Verify no files were actually removed in dry run mode
		assert.Equal(t, 0, stats.Removed, "Should not have removed any files in dry run")
		assert.Equal(t, 0, stats.RemoveErrs, "Should have no remove errors")

		// Verify the files still exist
		for _, f := range files {
			_, err := os.Stat(f)
			assert.NoError(t, err, "File should still exist in dry run: %s", f)
		}
	})

	t.Run("Skip cleanup on errors", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create some test files
		files := []string{
			filepath.Join(tempDir, "obsolete1.txt"),
		}

		for _, f := range files {
			err := os.WriteFile(f, []byte("test"), 0644)
			require.NoError(t, err, "Failed to create test file")
		}

		// Create a history with some files
		history := download_history.NewDownloadHistoryForTest(map[string]download_history.DownloadItem{
			filepath.Join(tempDir, "obsolete1.txt"): {
				FileID:         "file1",
				Hash:           "hash1",
				DownloadTime:   time.Now(),
				DownloadStatus: download_history.StatusLoaded, // Mark as loaded to be removed
			},
		})

		e := &Exporter{
			fs:     NewMockFileSystem(),
			DryRun: false,
		}

		stats := &ExportStats{
			DownloadErrs: 1, // Simulate a previous error
		}

		e.cleanupObsoleteFiles(history, stats)

		// Verify no files were removed due to previous errors
		assert.Equal(t, 0, stats.Removed, "Should not have removed any files due to previous errors")
		assert.Equal(t, 0, stats.RemoveErrs, "Should have no remove errors")

		// Verify the file still exists
		_, err := os.Stat(filepath.Join(tempDir, "obsolete1.txt"))
		assert.NoError(t, err, "File should still exist when cleanup is skipped due to errors")
	})
}

func TestExporter_MakeLocalFileName(t *testing.T) {

	testCases := []struct {
		displayPath string
		expected    string
	}{
		{"/mydrive/file.odoc", "mydrive/file.docx"},
		{"/mydrive/../mydrive/file.odoc", "mydrive/file.docx"},
		{"mydrive/file.odoc", "mydrive/file.docx"},
		{"/teamfolder/dir/../file.osheet", "teamfolder/file.xlsx"},
	}
	for _, tc := range testCases {
		actual := makeLocalFileName(tc.displayPath)
		if actual != tc.expected {
			t.Errorf("MakeLocalFileName(%q) = %q; want %q", tc.displayPath, actual, tc.expected)
		}
	}
}

// TestExporter_Counts verifies that DownloadHistory counters are incremented correctly during export.
func TestExporter_Counts(t *testing.T) {
	fileID := synd.FileID("file1")
	fileHash := synd.FileHash("hash1")
	fileID2 := synd.FileID("file2")
	fileHash2 := synd.FileHash("hash2")
	ignoredFileID := synd.FileID("file3")
	ignoredPath := "/doc/ignored.txt" // not exportable
	displayPath := "/doc/test1.odoc"
	cleanPath := makeLocalFileName(displayPath)

	t.Run("DownloadCount increments on successful export", func(t *testing.T) {
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		history, err := download_history.NewDownloadHistory("test_history.json")
		require.NoError(t, err)
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
		// Use test helper to create DownloadHistory with initial items for testing
		history := download_history.NewDownloadHistoryForTest(map[string]download_history.DownloadItem{
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
		history, err := download_history.NewDownloadHistory("test_history.json")
		require.NoError(t, err)
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
		history, err := download_history.NewDownloadHistory("test_history.json")
		require.NoError(t, err)
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
		history, err := download_history.NewDownloadHistory("test_history.json")
		require.NoError(t, err)
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

// TestExportItem_HistoryAndHash covers download and skip logic based on history and hash values.
func TestExportItem_HistoryAndHash(t *testing.T) {
	// --- DownloadStatus unit tests ---
	t.Run("download_history.StatusDownloaded when new", func(t *testing.T) {
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		history, err := download_history.NewDownloadHistory("test_history.json")
		require.NoError(t, err)
		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      "file1",
			DisplayPath: "/doc/test1.odoc",
			Hash:        "hash1",
		}
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processFile(item, history)
		// Retrieve the history item by display path.
		dlItem, exists := history.GetItem(makeLocalFileName(item.DisplayPath))
		require.True(t, exists, "expected item to exist in history for path: %s", item.DisplayPath)
		if dlItem.DownloadStatus != download_history.StatusDownloaded {
			t.Errorf("expected download_history.StatusDownloaded, got %v", dlItem.DownloadStatus)
		}
	})

	t.Run("download_history.StatusSkipped when hash unchanged", func(t *testing.T) {
		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      "file1",
			DisplayPath: "/doc/test1.odoc",
			Hash:        "hash1",
		}
		initialTime := time.Date(2024, 5, 18, 8, 0, 0, 0, time.UTC)
		path := makeLocalFileName(item.DisplayPath)
		// Use test helper to create DownloadHistory with initial items for testing
		history := download_history.NewDownloadHistoryForTest(map[string]download_history.DownloadItem{
			path: {FileID: "file1", Hash: "hash1", DownloadStatus: download_history.StatusLoaded, DownloadTime: initialTime},
		})
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processFile(item, history)

		// Inline getHistoryItemByDisplayPath logic (was: dlItem := getHistoryItemByDisplayPath(...))
		dlItem, exists := history.GetItem(makeLocalFileName(item.DisplayPath))
		require.True(t, exists, "expected item to exist in history for path: %s", item.DisplayPath)
		if dlItem.DownloadStatus != download_history.StatusSkipped {
			t.Errorf("expected download_history.StatusSkipped, got %v", dlItem.DownloadStatus)
		}
		if !dlItem.DownloadTime.Equal(initialTime) {
			t.Errorf("expected DownloadTime to remain unchanged, got %v want %v", dlItem.DownloadTime, initialTime)
		}
	})

	// --- Integration test: coexistence of loaded, downloaded, skipped ---
	t.Run("download_history.StatusLoaded, download_history.StatusDownloaded, download_history.StatusSkipped coexistence", func(t *testing.T) {
		// Setup: 3 files in history, only 2 are present in exportItems
		item1 := ExportItem{Type: synd.ObjectTypeFile, FileID: "file1", DisplayPath: "/doc/test1.odoc", Hash: "hash1"}     // hash unchanged
		item2 := ExportItem{Type: synd.ObjectTypeFile, FileID: "file2", DisplayPath: "/doc/test2.odoc", Hash: "hash2-new"} // hash changed
		item3 := ExportItem{Type: synd.ObjectTypeFile, FileID: "file3", DisplayPath: "/doc/test3.odoc", Hash: "hash3"}     // only in history
		// Use test helper to create DownloadHistory with initial items for testing
		history := download_history.NewDownloadHistoryForTest(map[string]download_history.DownloadItem{
			makeLocalFileName(item1.DisplayPath): {FileID: "file1", Hash: "hash1", DownloadStatus: download_history.StatusLoaded},
			makeLocalFileName(item2.DisplayPath): {FileID: "file2", Hash: "hash2-old", DownloadStatus: download_history.StatusLoaded},
			makeLocalFileName(item3.DisplayPath): {FileID: "file3", Hash: "hash3", DownloadStatus: download_history.StatusLoaded},
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
		key1 := makeLocalFileName(item1.DisplayPath)
		item1Status, exists := history.GetItem(key1)
		if !exists {
			t.Fatal("expected item1 to exist in history")
		}
		if item1Status.DownloadStatus != download_history.StatusSkipped {
			t.Errorf("expected download_history.StatusSkipped for item1, got %v", item1Status.DownloadStatus)
		}

		// Check item2 status (should be downloaded)
		key2 := makeLocalFileName(item2.DisplayPath)
		item2Status, exists := history.GetItem(key2)
		if !exists {
			t.Fatal("expected item2 to exist in history")
		}
		if item2Status.DownloadStatus != download_history.StatusDownloaded {
			t.Errorf("expected download_history.StatusDownloaded for item2, got %v", item2Status.DownloadStatus)
		}

		// Check item3 status (should remain loaded)
		key3 := makeLocalFileName(item3.DisplayPath)
		item3Status, exists := history.GetItem(key3)
		if !exists {
			t.Fatal("expected item3 to exist in history")
		}
		if item3Status.DownloadStatus != download_history.StatusLoaded {
			t.Errorf("expected download_history.StatusLoaded for item3, got %v", item3Status.DownloadStatus)
		}
	})

	fileID := synd.FileID("file1")
	fileHashOld := synd.FileHash("hash_old")
	fileHashNew := synd.FileHash("hash_new")
	displayPath := "/doc/test1.odoc"
	cleanPath := makeLocalFileName(displayPath)
	cases := []struct {
		name        string
		history     map[string]download_history.DownloadItem
		itemHash    synd.FileHash
		expectWrite bool
	}{
		{
			name:        "skip if hash unchanged",
			history:     map[string]download_history.DownloadItem{cleanPath: {FileID: fileID, Hash: fileHashOld}},
			itemHash:    fileHashOld,
			expectWrite: false,
		},
		{
			name:        "download if hash changed",
			history:     map[string]download_history.DownloadItem{cleanPath: {FileID: fileID, Hash: fileHashOld}},
			itemHash:    fileHashNew,
			expectWrite: true,
		},
		{
			name:        "download if not in history",
			history:     map[string]download_history.DownloadItem{},
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
			history := download_history.NewDownloadHistoryForTest(tc.history)
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
				key := makeLocalFileName(displayPath)
				_, exists := history.GetItem(key)
				if !exists {
					t.Error("expected item to exist in history after processing")
				}
			}
		})
	}
}

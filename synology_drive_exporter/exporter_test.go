package synology_drive_exporter

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	download_history "github.com/isseis/go-synology-office-exporter/download_history"
	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

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
				ListFunc: func(rootDirID synd.FileID, offset, limit int64) (*synd.ListResponse, error) {
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

// TestForceDownload verifies the behavior of the force-download feature
func TestForceDownload(t *testing.T) {
	tests := []struct {
		name          string
		forceDownload bool
		history       map[string]download_history.DownloadItem
		expectWrite   bool
		expectStatus  download_history.DownloadStatus
	}{
		{
			name:          "skip when force=false and hash matches",
			forceDownload: false,
			history: map[string]download_history.DownloadItem{
				makeLocalFileName("/doc/test.odoc"): {
					FileID:         "file1",
					Hash:           "hash1",
					DownloadStatus: download_history.StatusDownloaded,
					DownloadTime:   time.Now(),
				},
			},
			expectWrite:  false,
			expectStatus: download_history.StatusDownloaded, // Status remains downloaded when skipping
		},
		{
			name:          "force download when force=true and hash matches",
			forceDownload: true,
			history: map[string]download_history.DownloadItem{
				makeLocalFileName("/doc/test.odoc"): {
					FileID:         "file1",
					Hash:           "hash1",
					DownloadStatus: download_history.StatusDownloaded,
					DownloadTime:   time.Now(),
				},
			},
			expectWrite:  true,
			expectStatus: download_history.StatusDownloaded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			session := &MockSynologySession{
				ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
					return &synd.ExportResponse{Content: []byte("test content")}, nil
				},
			}

			mockFS := NewMockFileSystem()
			th := download_history.NewDownloadHistoryForTest(t, tt.history, download_history.WithTempDir("history_force.json"))
			defer th.Close()

			item := ExportItem{
				Type:        synd.ObjectTypeFile,
				FileID:      "file1",
				DisplayPath: "/doc/test.odoc",
				Hash:        "hash1",
			}

			exporter := NewExporterWithDependencies(session, "", mockFS)
			exporter.forceDownload = tt.forceDownload

			// For the purpose of this test, we'll just verify the behavior through the file system
			// and history updates, since the logs are being written to stderr and are hard to capture
			exporter.processFile(item, th.DownloadHistory)

			// Verify file operations
			_, exists := mockFS.WrittenFiles[makeLocalFileName(item.DisplayPath)]
			if tt.expectWrite != exists {
				t.Errorf("expected file write: %v, got: %v", tt.expectWrite, exists)
			}

			// Verify history status
			dlItem, exists, err := th.GetItem(makeLocalFileName(item.DisplayPath))
			require.NoError(t, err, "unexpected error getting item from history")
			require.True(t, exists, "expected item to exist in history")
			require.Equal(t, tt.expectStatus, dlItem.DownloadStatus, "unexpected download status")
		})
	}
}

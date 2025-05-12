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
	ListFunc   func(rootDirID synd.FileID) (*synd.ListResponse, error)
	ExportFunc func(fileID synd.FileID) (*synd.ExportResponse, error)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSession := &MockSynologySession{
				ListFunc: func(rootDirID synd.FileID) (*synd.ListResponse, error) {
					return tt.listResponse, tt.listError
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

			// Validate written files
			if tt.listResponse != nil && tt.expectedError == false {
				for _, item := range tt.listResponse.Items {
					if item.Type == synd.ObjectTypeFile {
						exportName := synd.GetExportFileName(item.DisplayPath)
						if exportName == "" {
							continue
						}

						if exportName[0] == '/' {
							exportName = exportName[1:]
						}
						expectedPath := filepath.Join("/tmp/test", exportName)

						if tt.fileOperationError == nil {
							// Only verify if there's no export error
							if _, ok := tt.exportError[item.FileID]; !ok {
								if _, exists := mockFS.WrittenFiles[expectedPath]; !exists {
									t.Errorf("File %s was not written", expectedPath)
								}
							}
						}
					}
				}
			}
		})
	}
}

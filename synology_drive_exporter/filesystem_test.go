package synology_drive_exporter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultFileSystem_CreateFile(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string // returns filename
		data      []byte
		dirPerm   os.FileMode
		filePerm  os.FileMode
		wantErr   bool
		checkFile bool
	}{
		{
			name: "successful file creation",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "subdir", "testfile.txt")
			},
			data:      []byte("test content"),
			dirPerm:   0755,
			filePerm:  0644,
			wantErr:   false,
			checkFile: true,
		},
		{
			name: "empty filename",
			setup: func(t *testing.T) string {
				return ""
			},
			data:     []byte("test"),
			dirPerm:  0755,
			filePerm: 0644,
			wantErr:  true,
		},
		{
			name: "fail when creating file with zero permission mask",
			setup: func(t *testing.T) string {
				// Try to create in a non-existent parent with no permissions
				return filepath.Join(t.TempDir(), "nonexistent", "testfile.txt")
			},
			data:     []byte("test"),
			dirPerm:  0000, // No permissions
			filePerm: 0644,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.setup(t)
			fs := &DefaultFileSystem{}

			err := fs.CreateFile(filename, tt.data, tt.dirPerm, tt.filePerm)

			if tt.wantErr {
				assert.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "unexpected error creating file")

			if tt.checkFile {
				// Verify file exists and has correct content
				content, err := os.ReadFile(filename)
				require.NoError(t, err, "failed to read created file")
				assert.Equal(t, tt.data, content, "file content mismatch")

				// Verify file permissions
				info, err := os.Stat(filename)
				require.NoError(t, err, "failed to get file info")
				assert.Equal(t, tt.filePerm, info.Mode().Perm(), "file permissions mismatch")

				// Verify parent directory permissions
				dir := filepath.Dir(filename)
				dirInfo, err := os.Stat(dir)
				require.NoError(t, err, "failed to get directory info")
				assert.True(t, dirInfo.IsDir(), "parent directory not created")
				assert.Equal(t, tt.dirPerm, dirInfo.Mode().Perm(), "directory permissions mismatch")
			}
		})
	}
}

func TestDefaultFileSystem_Remove(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string // returns filename
		wantErr func(e error) bool
	}{
		{
			name: "successful file removal",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filename := filepath.Join(dir, "testfile.txt")
				err := os.WriteFile(filename, []byte("test"), 0644)
				require.NoError(t, err, "failed to create test file")
				return filename
			},
			wantErr: nil,
		},
		{
			name: "non-existent file",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.txt")
			},
			wantErr: os.IsNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.setup(t)
			fs := &DefaultFileSystem{}

			err := fs.Remove(filename)

			if tt.wantErr != nil {
				assert.Error(t, err, "expected error")
				assert.True(t, tt.wantErr(err), "expected error")
			} else {
				assert.NoError(t, err, "unexpected error removing file")
				// Verify file no longer exists
				_, err = os.Stat(filename)
				assert.True(t, os.IsNotExist(err), "file should not exist after removal")
			}
		})
	}
}

package synology_drive_exporter

import (
	"errors"
	"os"
	"testing"
	"time"

	dh "github.com/isseis/go-synology-office-exporter/download_history"
	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
	"github.com/stretchr/testify/require"
)

var noDownloadItems = map[string]dh.DownloadItem{}

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
		th := dh.NewDownloadHistoryForTest(t, noDownloadItems)
		defer th.Close()
		history := th.DownloadHistory
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
		th := dh.NewDownloadHistoryForTest(t, map[string]dh.DownloadItem{
			cleanPath: {FileID: fileID, Hash: fileHash},
		})
		defer th.Close()
		history := th.DownloadHistory
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
		history, err := dh.NewDownloadHistory("test_history.json")
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
		history, err := dh.NewDownloadHistory("test_history.json")
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
		th := dh.NewDownloadHistoryForTest(t, noDownloadItems)
		defer th.Close()
		history := th.DownloadHistory
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
	t.Run("dh.StatusDownloaded when new", func(t *testing.T) {
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		th := dh.NewDownloadHistoryForTest(t, noDownloadItems, dh.WithTempDir("history.json"))
		defer th.Close()
		history := th.DownloadHistory

		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      "file1",
			DisplayPath: "/doc/test1.odoc",
			Hash:        "hash1",
		}
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processFile(item, history)
		// Retrieve the history item by display path.
		dlItem, exists, err := history.GetItem(makeLocalFileName(item.DisplayPath))
		require.NoError(t, err, "unexpected error getting item from history")
		require.True(t, exists, "expected item to exist in history for path: %s", item.DisplayPath)
		require.Equal(t, dh.StatusDownloaded, dlItem.DownloadStatus, "expected status to be downloaded")
	})

	t.Run("dh.StatusSkipped when hash unchanged", func(t *testing.T) {
		item := ExportItem{
			Type:        synd.ObjectTypeFile,
			FileID:      "file1",
			DisplayPath: "/doc/test1.odoc",
			Hash:        "hash1",
		}
		initialTime := time.Date(2024, 5, 18, 8, 0, 0, 0, time.UTC)
		path := makeLocalFileName(item.DisplayPath)
		// Use test helper to create DownloadHistory with initial items for testing
		th := dh.NewDownloadHistoryForTest(t, map[string]dh.DownloadItem{
			path: {FileID: "file1", Hash: "hash1", DownloadStatus: dh.StatusLoaded, DownloadTime: initialTime},
		}, dh.WithTempDir("history.json"))
		defer th.Close()
		history := th.DownloadHistory
		session := &MockSynologySession{
			ExportFunc: func(fid synd.FileID) (*synd.ExportResponse, error) {
				return &synd.ExportResponse{Content: []byte("file content")}, nil
			},
		}
		mockFS := NewMockFileSystem()
		exporter := NewExporterWithDependencies(session, "", mockFS)
		exporter.processFile(item, history)

		// Inline getHistoryItemByDisplayPath logic (was: dlItem := getHistoryItemByDisplayPath(...))
		dlItem, exists, err := history.GetItem(makeLocalFileName(item.DisplayPath))
		require.NoError(t, err, "unexpected error getting item from history")
		require.True(t, exists, "expected item to exist in history for path: %s", item.DisplayPath)
		if dlItem.DownloadStatus != dh.StatusSkipped {
			t.Errorf("expected dh.StatusSkipped, got %v", dlItem.DownloadStatus)
		}
		if !dlItem.DownloadTime.Equal(initialTime) {
			t.Errorf("expected DownloadTime to remain unchanged, got %v want %v", dlItem.DownloadTime, initialTime)
		}
	})

	// --- Integration test: coexistence of loaded, downloaded, skipped ---
	t.Run("dh.StatusLoaded, dh.StatusDownloaded, dh.StatusSkipped coexistence", func(t *testing.T) {
		// Setup: 3 files in history, only 2 are present in exportItems
		item1 := ExportItem{Type: synd.ObjectTypeFile, FileID: "file1", DisplayPath: "/doc/test1.odoc", Hash: "hash1"}     // hash unchanged
		item2 := ExportItem{Type: synd.ObjectTypeFile, FileID: "file2", DisplayPath: "/doc/test2.odoc", Hash: "hash2-new"} // hash changed
		item3 := ExportItem{Type: synd.ObjectTypeFile, FileID: "file3", DisplayPath: "/doc/test3.odoc", Hash: "hash3"}     // only in history
		// Use test helper to create DownloadHistory with initial items for testing
		th := dh.NewDownloadHistoryForTest(t, map[string]dh.DownloadItem{
			makeLocalFileName(item1.DisplayPath): {FileID: "file1", Hash: "hash1", DownloadStatus: dh.StatusLoaded},
			makeLocalFileName(item2.DisplayPath): {FileID: "file2", Hash: "hash2-old", DownloadStatus: dh.StatusLoaded},
			makeLocalFileName(item3.DisplayPath): {FileID: "file3", Hash: "hash3", DownloadStatus: dh.StatusLoaded},
		}, dh.WithTempDir("history.json"))
		defer th.Close()
		history := th.DownloadHistory
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
		item1Status, exists, err := history.GetItem(key1)
		require.NoError(t, err, "unexpected error getting item1 from history")
		if !exists {
			t.Fatal("expected item1 to exist in history")
		}
		if item1Status.DownloadStatus != dh.StatusSkipped {
			t.Errorf("expected dh.StatusSkipped for item1, got %v", item1Status.DownloadStatus)
		}

		// Check item2 status (should be downloaded)
		key2 := makeLocalFileName(item2.DisplayPath)
		item2Status, exists, err := history.GetItem(key2)
		require.NoError(t, err, "unexpected error getting item2 from history")
		if !exists {
			t.Fatal("expected item2 to exist in history")
		}
		if item2Status.DownloadStatus != dh.StatusDownloaded {
			t.Errorf("expected dh.StatusDownloaded for item2, got %v", item2Status.DownloadStatus)
		}

		// Check item3 status (should remain loaded)
		key3 := makeLocalFileName(item3.DisplayPath)
		item3Status, exists, err := history.GetItem(key3)
		require.NoError(t, err, "unexpected error getting item3 from history")
		if !exists {
			t.Fatal("expected item3 to exist in history")
		}
		if item3Status.DownloadStatus != dh.StatusLoaded {
			t.Errorf("expected dh.StatusLoaded for item3, got %v", item3Status.DownloadStatus)
		}
	})

	fileID := synd.FileID("file1")
	fileHashOld := synd.FileHash("hash_old")
	fileHashNew := synd.FileHash("hash_new")
	displayPath := "/doc/test1.odoc"
	cleanPath := makeLocalFileName(displayPath)
	cases := []struct {
		name        string
		history     map[string]dh.DownloadItem
		itemHash    synd.FileHash
		expectWrite bool
	}{
		{
			name:        "skip if hash unchanged",
			history:     map[string]dh.DownloadItem{cleanPath: {FileID: fileID, Hash: fileHashOld}},
			itemHash:    fileHashOld,
			expectWrite: false,
		},
		{
			name:        "download if hash changed",
			history:     map[string]dh.DownloadItem{cleanPath: {FileID: fileID, Hash: fileHashOld}},
			itemHash:    fileHashNew,
			expectWrite: true,
		},
		{
			name:        "download if not in history",
			history:     noDownloadItems,
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
			th := dh.NewDownloadHistoryForTest(t, tc.history)
			defer th.Close()
			history := th.DownloadHistory
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
				_, exists, err := history.GetItem(key)
				require.NoError(t, err, "unexpected error checking if item exists in history")
				if !exists {
					t.Error("expected item to exist in history after processing")
				}
			}
		})
	}
}

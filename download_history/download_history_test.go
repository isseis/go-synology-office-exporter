//go:build !integration
// +build !integration

package download_history

import (
	"testing"
	"time"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

func TestDownloadHistoryBasic(t *testing.T) {
	history, err := NewDownloadHistory("test_history.json")
	if err != nil {
		t.Fatalf("failed to create history: %v", err)
	}

	item := DownloadItem{
		FileID:         synd.FileID("id1"),
		Hash:           synd.FileHash("hash1"),
		DownloadTime:   time.Now(),
		DownloadStatus: StatusLoaded,
	}

	history.SetItem("file1", item)
	got, ok := history.Items["file1"]
	if !ok || got.FileID != item.FileID {
		t.Errorf("SetItem or Items failed: got %+v", got)
	}
}

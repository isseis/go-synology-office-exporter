package download_history

import (
	"time"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// DownloadStatus represents the state of a DownloadItem (enum-like string type).
type DownloadStatus string

const (
	StatusLoaded     DownloadStatus = "loaded"
	StatusDownloaded DownloadStatus = "downloaded"
	StatusSkipped    DownloadStatus = "skipped"
)

// DownloadItem holds information about a downloaded or tracked file, including its status.
type DownloadItem struct {
	FileID         synd.FileID
	Hash           synd.FileHash
	DownloadTime   time.Time
	DownloadStatus DownloadStatus
}

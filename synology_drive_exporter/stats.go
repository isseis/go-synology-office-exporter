package synology_drive_exporter

import (
	"fmt"

	dh "github.com/isseis/go-synology-office-exporter/download_history"
)

// ExportStats holds the statistics of the export operation
type ExportStats struct {
	Downloaded   int // Number of successfully downloaded files
	Skipped      int // Number of skipped files (already up-to-date)
	Ignored      int // Number of ignored files (not exportable)
	Removed      int // Number of successfully removed files
	DownloadErrs int // Number of errors occurred during download
	RemoveErrs   int // Number of errors occurred during removal
}

// String returns a string representation of the export statistics
func (s ExportStats) String() string {
	return fmt.Sprintf("downloaded=%d, skipped=%d, ignored=%d, removed=%d, download_errors=%d, remove_errors=%d",
		s.Downloaded, s.Skipped, s.Ignored, s.Removed, s.DownloadErrs, s.RemoveErrs)
}

func (s *ExportStats) IncrementRemoved() {
	s.Removed++
}

// IncrementDownloadErrs increments the download error count
func (s *ExportStats) IncrementDownloadErrs() {
	s.DownloadErrs++
}

// IncrementRemoveErrs increments the removal error count
func (s *ExportStats) IncrementRemoveErrs() {
	s.RemoveErrs++
}

// TotalErrs returns the total number of errors
func (s *ExportStats) TotalErrs() int {
	return s.DownloadErrs + s.RemoveErrs
}

func toExportStats(stats dh.ExportStats) ExportStats {
	return ExportStats{
		Downloaded:   stats.Downloaded,
		Skipped:      stats.Skipped,
		Ignored:      stats.Ignored,
		Removed:      0,
		DownloadErrs: stats.Errors,
		RemoveErrs:   0,
	}
}

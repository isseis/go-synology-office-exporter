package synology_drive_exporter

import (
	"fmt"
	"os"

	dh "github.com/isseis/go-synology-office-exporter/download_history"
)

// removeFile removes the specified file from the filesystem.
// In DryRun mode, it only logs the operation without actually removing the file.
func (e *Exporter) removeFile(path string) error {
	if e.IsDryRun() {
		e.getLogger().Debug("Dry run: would remove file", "path", path)
		return nil
	}

	err := e.fs.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			e.getLogger().Debug("File already removed", "path", path)
			return nil
		}
		return fmt.Errorf("failed to remove file %s: %w", path, err)
	}
	e.getLogger().Debug("File removed successfully", "path", path)
	return nil
}

// cleanupObsoleteFiles removes files that exist in history but not in the current export
// It skips cleanup if there were any errors during the export process
func (e *Exporter) cleanupObsoleteFiles(history *dh.DownloadHistory, stats *ExportStats) error {
	if stats.TotalErrs() > 0 {
		e.getLogger().Info("Skipping cleanup due to previous errors")
		return nil
	}

	obsoletePaths, err := history.GetObsoleteItems()
	if err != nil {
		e.getLogger().Error("Failed to get obsolete items", "error", err)
		return err
	}
	for _, path := range obsoletePaths {
		if err := e.removeFile(path); err != nil {
			stats.IncrementRemoveErrs() // Always count errors
			e.getLogger().Error("Failed to remove obsolete file", "path", path, "error", err)
		} else {
			stats.IncrementRemoved() // Always count successful removals
		}
	}
	return nil
}

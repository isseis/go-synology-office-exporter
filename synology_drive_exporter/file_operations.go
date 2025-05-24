package synology_drive_exporter

import (
	"fmt"
	"log"
	"os"

	dh "github.com/isseis/go-synology-office-exporter/download_history"
)

// removeFile removes the specified file from the filesystem.
// In DryRun mode, it only logs the operation without actually removing the file.
func (e *Exporter) removeFile(path string) error {
	if e.IsDryRun() {
		log.Printf("[DRY RUN] Would remove file: %s", path)
		return nil
	}

	err := e.FileSystemOperations.RemoveFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("File already removed: %s", path)
			return nil
		}
		return fmt.Errorf("failed to remove file %s: %w", path, err)
	}
	log.Printf("Removed file: %s", path)
	return nil
}

// cleanupObsoleteFiles removes files that exist in history but not in the current export
// It skips cleanup if there were any errors during the export process
func (e *Exporter) cleanupObsoleteFiles(history *dh.DownloadHistory, stats *ExportStats) error {
	if stats.TotalErrs() > 0 {
		log.Println("Skipping cleanup due to previous errors")
		return nil
	}

	obsoletePaths, err := history.GetObsoleteItems()
	if err != nil {
		log.Printf("Error getting obsolete items: %v", err)
		return err
	}
	for _, path := range obsoletePaths {
		if err := e.removeFile(path); err != nil {
			stats.IncrementRemoveErrs() // Always count errors
			log.Printf("Error removing file: %v", err)
		} else {
			stats.IncrementRemoved() // Always count successful removals
		}
	}
	return nil
}

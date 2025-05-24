package download_history

// ExportStats holds the statistics of the export operation.
type ExportStats struct {
	Downloaded int // Number of successfully downloaded files
	Skipped    int // Number of skipped files (already up-to-date)
	Ignored    int // Number of ignored files (not exportable)
	Errors     int // Number of errors occurred
}

package synology_drive_exporter

import (
	"fmt"
)

// DownloadHistoryOperationError represents an error during a download history operation.
type DownloadHistoryOperationError struct {
	Op  string // operation description
	Err error  // underlying error
}

func (e *DownloadHistoryOperationError) Error() string {
	return fmt.Sprintf("download history operation error [%s]: %v", e.Op, e.Err)
}

func (e *DownloadHistoryOperationError) Unwrap() error {
	return e.Err
}

// ExportFileWriteError represents an error that occurred while writing an export file.
type ExportFileWriteError struct {
	Op  string // operation description
	Err error  // underlying error
}

func (e ExportFileWriteError) Error() string {
	return fmt.Sprintf("failed to write export file [%s]: %v", e.Op, e.Err)
}

func (e ExportFileWriteError) Unwrap() error {
	return e.Err
}

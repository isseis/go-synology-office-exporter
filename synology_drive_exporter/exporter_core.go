package synology_drive_exporter

import (
	"fmt"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// Logger defines the interface for logging operations within the exporter.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	FlushWebhook() error
}

// Exporter handles exporting files from Synology Drive, maintaining download history and file system abstraction.
type Exporter struct {
	session     SessionInterface
	downloadDir string // Directory where downloaded files will be saved
	fs          FileSystemOperations
	logger      Logger // Logger for structured logging

	// dryRun controls whether file operations are performed. Immutable after construction.
	// Default is false.
	dryRun bool

	// forceDownload controls whether to re-download files even if they exist and have matching hashes.
	// Default is false.
	forceDownload bool
}

// ExporterOption defines a function type to set options for Exporter.
// Use WithDryRun and similar helpers to specify runtime options.
type ExporterOption func(*Exporter)

// WithDryRun sets the dryRun option for Exporter.
func WithDryRun(dryRun bool) ExporterOption {
	return func(e *Exporter) {
		e.dryRun = dryRun
	}
}

// WithForceDownload sets the forceDownload option for Exporter.
// When true, files will be re-downloaded even if they exist and have matching hashes.
func WithForceDownload(force bool) ExporterOption {
	return func(e *Exporter) {
		e.forceDownload = force
	}
}

// WithLogger sets the logger for Exporter.
// If not set, a fallback logger will be used for backward compatibility.
func WithLogger(log Logger) ExporterOption {
	return func(e *Exporter) {
		e.logger = log
	}
}

// IsDryRun returns true if the exporter is in dry-run mode.
func (e *Exporter) IsDryRun() bool {
	return e.dryRun
}

// getLogger returns the logger, falling back to a default logger if none is set.
func (e *Exporter) getLogger() Logger {
	if e.logger != nil {
		return e.logger
	}
	return &fallbackLogger{}
}

// GetLogger returns the logger instance for testing purposes.
// This method is intended for testing and debugging only.
func (e *Exporter) GetLogger() Logger {
	return e.getLogger()
}

// fallbackLogger provides a backward-compatible logging implementation.
type fallbackLogger struct{}

func (f *fallbackLogger) Debug(msg string, args ...any) {
	fmt.Printf("[DEBUG] %s\n", formatLogMessage(msg, args...))
}

func (f *fallbackLogger) Info(msg string, args ...any) {
	fmt.Printf("[INFO] %s\n", formatLogMessage(msg, args...))
}

func (f *fallbackLogger) Warn(msg string, args ...any) {
	fmt.Printf("[WARN] %s\n", formatLogMessage(msg, args...))
}

func (f *fallbackLogger) Error(msg string, args ...any) {
	fmt.Printf("[ERROR] %s\n", formatLogMessage(msg, args...))
}

func (f *fallbackLogger) FlushWebhook() error {
	return nil
}

// formatLogMessage formats the log message with key-value pairs.
// It concatenates the message with additional key-value pairs provided in args.
// In case of an odd number of args, the last one is ignored.
func formatLogMessage(msg string, args ...any) string {
	result := msg
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			result += fmt.Sprintf(" %v=%v", args[i], args[i+1])
		}
	}
	return result
}

// NewExporter constructs an Exporter with a real Synology session and the specified download directory. If downloadDir is empty, the current directory is used.
// Additional runtime options can be specified via ExporterOption(s), such as WithDryRun.
func NewExporter(username string, password string, base_url string, downloadDir string, opts ...ExporterOption) (*Exporter, error) {
	session, err := synd.NewSynologySession(username, password, base_url)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	if err = session.Login(); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	exporter := NewExporterWithDependencies(session, downloadDir, &DefaultFileSystem{}, opts...)
	return exporter, nil
}

// NewExporterWithDependencies constructs an Exporter with injected dependencies for session, download directory, and file system. Intended for testing and advanced use.
// Additional runtime options can be specified via ExporterOption(s), such as WithDryRun.
func NewExporterWithDependencies(session SessionInterface, downloadDir string, fs FileSystemOperations, opts ...ExporterOption) *Exporter {
	e := &Exporter{
		session:     session,
		downloadDir: downloadDir,
		fs:          fs,
		dryRun:      false, // default
		logger:      nil,   // will use fallback logger if not set
	}
	// Apply additional runtime options.
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// ExportMyDrive exports convertible files from the user's Synology Drive, using download history to avoid duplicates.
func (e *Exporter) ExportMyDrive() (ExportStats, error) {
	return e.ExportRootsWithHistory(
		[]synd.FileID{synd.MyDrive},
		"mydrive_history.json",
	)
}

// ExportTeamFolder exports convertible files from all team folders, using download history to avoid duplicates.
func (e *Exporter) ExportTeamFolder() (ExportStats, error) {
	teamFolders, err := teamFoldersAll(e.session)
	if err != nil {
		return ExportStats{}, err
	}
	var rootIDs []synd.FileID
	for _, item := range teamFolders {
		rootIDs = append(rootIDs, item.FileID)
	}
	return e.ExportRootsWithHistory(
		rootIDs,
		"team_folder_history.json",
	)
}

// ExportSharedWithMe exports convertible files and directories shared with the user, using download history to avoid duplicates.
func (e *Exporter) ExportSharedWithMe() (ExportStats, error) {
	sharedItems, err := sharedWithMeAll(e.session)
	if err != nil {
		return ExportStats{}, err
	}
	var exportItems []ExportItem
	for _, item := range sharedItems {
		exportItems = append(exportItems, newExportItem(item))
	}
	return e.exportItemsWithHistory(exportItems, "shared_with_me_history.json")
}

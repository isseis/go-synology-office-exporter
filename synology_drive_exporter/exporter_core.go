package synology_drive_exporter

import (
	"fmt"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// Exporter handles exporting files from Synology Drive, maintaining download history and file system abstraction.
type Exporter struct {
	session     SessionInterface
	downloadDir string // Directory where downloaded files will be saved
	fs          FileSystemOperations

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

// IsDryRun returns true if the exporter is in dry-run mode.
func (e *Exporter) IsDryRun() bool {
	return e.dryRun
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

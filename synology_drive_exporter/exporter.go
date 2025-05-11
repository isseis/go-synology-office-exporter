package synology_drive_exporter

import (
	"fmt"
	"os"
	"path/filepath"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// Exporter provides functionality to export files from Synology Drive.
type Exporter struct {
	session     *synd.SynologySession
	downloadDir string // Directory where downloaded files will be saved
}

// NewExporter creates a new Exporter instance with the default download directory (current directory).
func NewExporter(username string, password string, base_url string) (*Exporter, error) {
	session, err := synd.NewSynologySession(username, password, base_url)
	if err != nil {
		return nil, err
	}
	if err = session.Login(); err != nil {
		return nil, err
	}

	exporter := &Exporter{
		session:     session,
		downloadDir: ".", // Default to current directory
	}
	return exporter, nil
}

// NewExporterWithDownloadDir creates a new Exporter with the specified download directory.
func NewExporterWithDownloadDir(username string, password string, base_url string, downloadDir string) (*Exporter, error) {
	session, err := synd.NewSynologySession(username, password, base_url)
	if err != nil {
		return nil, err
	}
	if err = session.Login(); err != nil {
		return nil, err
	}

	exporter := &Exporter{
		session:     session,
		downloadDir: downloadDir,
	}
	return exporter, nil
}

// ExportMyDrive exports all convertible files from the user's Synology Drive
// and saves them to the download directory.
func (e *Exporter) ExportMyDrive() error {
	list, err := e.session.List(synd.MyDrive)
	if err != nil {
		return err
	}

	for _, item := range list.Items {
		if item.Type == synd.ObjectTypeFile {
			exportName := synd.GetExportFileName(item.DisplayPath)
			if exportName == "" {
				continue
			}
			fmt.Printf("Exporting file: %s\n", exportName)

			// Export the file
			resp, err := e.session.Export(item.FileID)
			if err != nil {
				fmt.Printf("Failed to export %s: %v\n", exportName, err)
				continue
			}

			// Save the file locally
			// TODO: Keep the original directory structure
			downloadPath := filepath.Join(e.downloadDir, filepath.Base(exportName))
			if err := os.WriteFile(downloadPath, resp.Content, 0644); err != nil {
				return fmt.Errorf("failed to save file %s: %w", downloadPath, err)
			}
			fmt.Printf("Saved to: %s\n", downloadPath)
		}
	}
	return nil
}

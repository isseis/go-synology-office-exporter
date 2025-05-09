package synology_drive_exporter

import (
	"fmt"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

type Exporter struct {
	session *synd.SynologySession
}

func NewExporter(username string, password string, base_url string) (*Exporter, error) {
	session, err := synd.NewSynologySession(username, password, base_url)
	if err != nil {
		return nil, err
	}
	if err = session.Login(); err != nil {
		return nil, err
	}

	exporter := &Exporter{
		session: session,
	}
	return exporter, nil
}

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
		}
	}
	// Implement the logic to export MyDrive data
	return nil
}

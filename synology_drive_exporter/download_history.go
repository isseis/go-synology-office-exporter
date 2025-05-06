package synology_drive_exporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const HISTORY_VERSION = 2
const HISTORY_MAGIC = "SYNOLOGY_OFFICE_EXPORTER"

type DownloadHistory struct{}

type jsonHeader struct {
	Version int    `json:"version"`
	Magic   string `json:"magic"`
	Created string `json:"created"`
}

type jsonDownloadItem struct {
	Location     string `json:"location"`
	FileID       string `json:"file_id"`
	Hash         string `json:"hash"`
	DownloadTime string `json:"download_time"`
}

type jsonDownloadHistory struct {
	Header jsonHeader         `json:"header"`
	Items  []jsonDownloadItem `json:"items"`
}

func NewDownloadHistory() *DownloadHistory {
	return &DownloadHistory{}
}

func (json *jsonHeader) validate() error {
	if json.Version != HISTORY_VERSION {
		return DownloadHistoryParseError(fmt.Sprintf("unsupported version: %d", json.Version))
	}
	if json.Magic != HISTORY_MAGIC {
		return DownloadHistoryParseError(fmt.Sprintf("invalid magic: %s", json.Magic))
	}
	return nil
}

func (json *jsonDownloadHistory) validate() error {
	if err := json.Header.validate(); err != nil {
		return err
	}
	return nil
}

func (d *DownloadHistory) Load(r io.Reader) error {
	content, err := io.ReadAll(r)
	if err != nil {
		return DownloadHistoryFileError(err.Error())
	}

	var history jsonDownloadHistory
	if err := json.Unmarshal(content, &history); err != nil {
		return DownloadHistoryParseError(err.Error())
	}

	if err := history.validate(); err != nil {
		return err
	}

	// TODO: Store the parsed history in the DownloadHistory struct
	// This part depends on how you want to use the data in the DownloadHistory struct
	return nil
}

func (d *DownloadHistory) LoadFile(path string) error {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return DownloadHistoryFileError(err.Error())
	}
	defer file.Close()
	return d.Load(file)
}

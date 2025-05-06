package synology_drive_exporter

import "strconv"

type DownloadHistoryFileError string

func (e DownloadHistoryFileError) Error() string {
	return "failed to read download history file: " + strconv.Quote(string(e))
}

type DownloadHistoryParseError string

func (e DownloadHistoryParseError) Error() string {
	return "failed to parse download history JSON: " + strconv.Quote(string(e))
}

package synology_drive_exporter

import "strconv"

type DownloadHistoryFileReadError string

func (e DownloadHistoryFileReadError) Error() string {
	return "failed to read download history file: " + strconv.Quote(string(e))
}

type DownloadHistoryFileWriteError string

func (e DownloadHistoryFileWriteError) Error() string {
	return "failed to write download history file: " + strconv.Quote(string(e))
}

type DownloadHistoryParseError string

func (e DownloadHistoryParseError) Error() string {
	return "failed to parse download history JSON: " + strconv.Quote(string(e))
}

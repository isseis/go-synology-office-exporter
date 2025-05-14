package synology_drive_exporter

import "strconv"

type DownloadHistoryFileError string

func (e DownloadHistoryFileError) Error() string {
	return "download history file error: " + strconv.Quote(string(e))
}

type DownloadHistoryFileIsNotFoundError string

func (e DownloadHistoryFileIsNotFoundError) Error() string {
	return "download history file is not found: " + strconv.Quote(string(e))
}

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

type ExportFileWriteError string

func (e ExportFileWriteError) Error() string {
	return "failed to write export file: " + strconv.Quote(string(e))
}

# Synology Office Exporter (Go)

![Go](https://img.shields.io/badge/go-1.24+-blue.svg)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](https://opensource.org/licenses/MIT)
![Contributions welcome](https://img.shields.io/badge/contributions-welcome-orange.svg)

A command-line tool to export documents from Synology Drive/Office to local files. Written in Go for efficiency, reliability, and easy deployment. This tool allows you to export documents from your Synology Drive, Team Folders, and Shared With Me locations to your local machine.

## Features

- Export Synology Office documents (Spreadsheet, Document, Slides) to Microsoft Office formats (e.g., xlsx, docx, pptx)
- Supports multiple export sources:
  - Personal Drive (My Drive)
  - Team Folders
  - Shared With Me
- Download history management to avoid duplicate exports
- Dry run mode to preview changes without downloading
- Comprehensive logging and error reporting
- CLI interface for automation and scripting
- Written in Go for cross-platform binaries and performance

## Requirements

- Go 1.24+ (for building from source)
- Synology NAS with Drive/Office enabled
- Network access to your Synology NAS

## Installation

### Download Pre-built Binary

Not available yet.

### Build from Source

```sh
git clone https://github.com/isseis/go-synology-office-exporter.git
cd go-synology-office-exporter
go build -o synology-office-exporter ./cmd/export
```

## Usage

### Environment Variables (Recommended)

It's recommended to provide sensitive credentials via environment variables to avoid leaking them in shell history or process lists.

```sh
export SYNOLOGY_NAS_URL=https://your-nas-address:port
export SYNOLOGY_NAS_USER=your_username
export SYNOLOGY_NAS_PASS=your_password
export SYNOLOGY_DOWNLOAD_DIR=./exports  # Optional, defaults to current directory
```

Then run:

```sh
./synology-office-exporter
```

### Command-Line Options

```
Usage of synology-office-exporter:
  -dry-run
        If set, perform a dry run (no file downloads, only show statistics)
  -force-download
        If set, re-download files even if they exist and have matching hashes
  -output string
        Directory to save downloaded files (can be set via env SYNOLOGY_DOWNLOAD_DIR)
  -pass string
        Synology NAS password (can be set via env SYNOLOGY_NAS_PASS)
  -sources string
        Comma-separated list of sources to export (mydrive,teamfolder,shared) (default "mydrive,teamfolder,shared")
  -url string
        Synology NAS URL (can be set via env SYNOLOGY_NAS_URL)
  -user string
        Synology NAS username (can be set via env SYNOLOGY_NAS_USER)
```

### Logger Environment Variables

The tool supports various logging configurations:

- `LOG_LEVEL`: Set log level (debug, info, warn, error) - default: info
- `LOG_WEBHOOK_URL`: Webhook URL for sending logs
- `APP_NAME`: Application name for logging
- `ENV`: Environment (development, staging, production)

#### Log Level Details:
- `debug`: Detailed processing information (file-by-file operations)
- `info`: Important operational information (start/completion messages, statistics)
- `warn`: Non-critical warnings and issues
- `error`: Error conditions that prevent processing

## Examples

### Basic Export

Export all documents to the default directory (current directory):

```sh
./synology-office-exporter -url https://your-nas:5001 -user your_username -pass your_password
```

### Export to Specific Directory

```sh
./synology-office-exporter -output ./synology_exports
```

### Dry Run (No Downloads)

Preview what would be downloaded without making any changes:

```sh
./synology-office-exporter -dry-run
```

### Export Specific Sources

Export only from My Drive and Team Folders:

```sh
./synology-office-exporter -sources mydrive,teamfolder
```

## Download History

The tool maintains history files to avoid re-downloading already exported documents:

- `mydrive_history.json` - Tracks exported files from My Drive
- `team_folder_history.json` - Tracks exported files from Team Folders
- `shared_with_me_history.json` - Tracks exported files from Shared With Me

To force a full re-export, delete or rename the appropriate history file(s).

## Security

- Credentials are only used for API requests and are not stored
- Use an application-specific account with minimal permissions
- Avoid using your main admin account
- Consider using environment variables or `.env` files for credentials
- All API communication is encrypted with HTTPS

## License

[MIT License](https://opensource.org/licenses/MIT)

## Acknowledgements

- [Synology Drive API](https://github.com/zbjdonald/synology-drive-api) - Inspired by this project for communication with the Synology Drive API

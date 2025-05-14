# Synology Office Exporter (Go)

![Go](https://img.shields.io/badge/go-1.24+-blue.svg)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](https://opensource.org/licenses/MIT)
![Contributions welcome](https://img.shields.io/badge/contributions-welcome-orange.svg)

A command-line tool to export documents from Synology Drive/Office to local files. Written in Go for efficiency, reliability, and easy deployment.

## Features

- Export Synology Office documents (Spreadsheet, Document, Slides) to Microsoft Office formats (e.g., xlsx, docx, pptx).
- Supports Synology Drive API authentication.
- Download history management to avoid duplicate exports (experimental; only MyDrive is supported).
- CLI interface for automation and scripting.
- Written in Go for cross-platform binaries and performance.

## Requirements

- Go 1.24+ (for building from source)
- Synology NAS with Drive/Office enabled
- API access enabled on Synology NAS

## Installation

### Download Pre-built Binary

Not available yet.

### Build from Source

```sh
git clone https://github.com/isseis/go-synology-office-exporter.git
cd go-synology-office-exporter
go build -o synology-office-exporter ./cmd/synology-office-exporter
```

## Usage

It is strongly recommended to provide sensitive credentials via environment variables to avoid leaking them in shell history or process lists.

### Environment Variables

Set the following variables before running the tool:

```sh
export SYNOLOGY_HOST=<SYNOLOGY_HOST>
export SYNOLOGY_USERNAME=<USERNAME>
export SYNOLOGY_PASSWORD=<PASSWORD>
```

Then run:

```sh
./synology-office-exporter \
  --output <OUTPUT_DIR> \
```

### Command-Line Options (Not Recommended for Credentials)

You may also provide `--host`, `--username`, and `--password` as command-line options, but this is discouraged for security reasons (these values may be visible to other local users via shell history or process list).

## Options

- `--host`         : Synology NAS hostname or IP address (can be set via env `SYNOLOGY_HOST`)
- `--username`     : Synology account username (can be set via env `SYNOLOGY_USERNAME`)
- `--password`     : Synology account password (can be set via env `SYNOLOGY_PASSWORD`)
- `--output`       : Local directory to save exported files (required)

## Example

Export all documents from MyDrive:

```sh
export SYNOLOGY_HOST=192.168.1.10
export SYNOLOGY_USERNAME=admin
export SYNOLOGY_PASSWORD=secret

./synology-office-exporter \
  --output ./exports \
```

## Download History

The tool maintains a history file (default: `mydrive_history.json`) to avoid re-downloading already exported documents. Delete or rename this file to force a full re-export.

## Security

- Credentials are only used for API requests and not stored.
- Use an application-specific account with minimal permissions.
- Avoid using your main admin account.

## License

[MIT License](https://opensource.org/licenses/MIT)

## Acknowledgements

- [Synology Drive API](https://github.com/zbjdonald/synology-drive-api) - Inspired by this project for communication with the Synology Drive API

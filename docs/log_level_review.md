# Log Level Review: Implementation Details

## Overview

This document describes the review and optimization of log levels for the Synology Office Exporter.

## Problem Statement

In the previous log output, many detailed processing information were output at the `info` level, resulting in verbose logs during production use.

## Changes

### 1. Default Log Level Change
- **Before**: `warn` (important information not displayed)
- **After**: `info` (appropriate balance)

### 2. Individual Log Message Level Adjustments

#### Messages Changed to Debug Level
The following messages were changed from `info` to `debug`:

**export_processor.go**:
- `"Skipping non-exportable file"` - Detailed file processing information
- `"Dry run: would export file"` - Detailed dry run information
- `"Exporting file"` - Detailed file processing information
- `"File exported successfully"` - Detailed file processing information

**file_operations.go**:
- `"Dry run: would remove file"` - Detailed dry run information
- `"File already removed"` - Detailed state information
- `"File removed successfully"` - Detailed processing information

### 3. Log Level Usage Guidelines

| Level | Purpose | Examples |
|-------|---------|----------|
| **Debug** | Detailed processing information, individual file operations | File processing, dry run details |
| **Info** | Important operational information, statistics | Start/completion messages, statistics |
| **Warn** | Non-fatal warnings | History update failures, retryable errors |
| **Error** | Error conditions | File processing failures, system errors |

## Operational Impact

### Default Setting (info level)
Information displayed during execution:
- Application start/completion
- Export statistics
- Important processing decisions
- Errors and warnings

Information not displayed:
- Individual file processing details
- Dry run detailed operations

### When Using Debug Level
All processing details are displayed, useful for troubleshooting.

## Configuration

```bash
# Standard usage (info level)
export LOG_LEVEL=info
./synology-office-exporter

# When detailed debug information is needed
export LOG_LEVEL=debug
./synology-office-exporter

# Minimal logging (warn level)
export LOG_LEVEL=warn
./synology-office-exporter
```

## Changed Files

1. `logger/config_loader.go` - Changed default level from `warn` → `info`
2. `synology_drive_exporter/export_processor.go` - Changed 4 messages from `info` → `debug`
3. `synology_drive_exporter/file_operations.go` - Changed 3 messages from `info` → `debug`
4. `README.md` - Updated log configuration documentation

## Backward Compatibility

- If the level is explicitly specified with the `LOG_LEVEL` environment variable, behavior is unchanged
- If specified with the `-log-level` command line flag, behavior is unchanged
- Only the default behavior has been changed

## Testing

All existing tests have been verified to pass successfully. Log level changes only affect log output and do not impact application functionality.

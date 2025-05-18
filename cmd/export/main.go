package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	syndexp "github.com/isseis/go-synology-office-exporter/synology_drive_exporter"
)

// sourceType represents the type of export source.
type sourceType string

// Valid source types
const (
	sourceMyDrive    sourceType = "mydrive"
	sourceTeamFolder sourceType = "teamfolder"
	sourceShared     sourceType = "shared"
)

// defaultSources returns all available source types.
func defaultSources() []sourceType {
	return []sourceType{sourceMyDrive, sourceTeamFolder, sourceShared}
}

// parseSources parses a comma-separated string of source types.
// Returns a slice of valid source types and any error encountered.
func parseSources(s string) ([]sourceType, error) {
	if s == "" {
		return defaultSources(), nil
	}

	parts := strings.Split(s, ",")
	sources := make([]sourceType, 0, len(parts))
	seen := make(map[sourceType]bool)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		var st sourceType
		switch part {
		case string(sourceMyDrive):
			st = sourceMyDrive
		case string(sourceTeamFolder):
			st = sourceTeamFolder
		case string(sourceShared):
			st = sourceShared
		default:
			return nil, fmt.Errorf("invalid source type: %s", part)
		}
		if !seen[st] {
			sources = append(sources, st)
			seen[st] = true
		}
	}

	if len(sources) == 0 {
		return defaultSources(), nil
	}

	return sources, nil
}

const Version = "0.1.0"

func init() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, relying on environment variables")
	}
}

func main() {
	fmt.Println("Starting Synology Office Exporter...")

	// Define command-line flags
	userFlag := flag.String("user", "", "Synology NAS username")
	passFlag := flag.String("pass", "", "Synology NAS password")
	urlFlag := flag.String("url", "", "Synology NAS URL")
	downloadDirFlag := flag.String("output", "", "Directory to save downloaded files")
	sourcesFlag := flag.String("sources", "mydrive,teamfolder,shared", "Comma-separated list of sources to export (mydrive,teamfolder,shared)")
	// dry_run: If true, performs a dry run (no files are downloaded or written, only statistics are shown)
	dryRunFlag := flag.Bool("dry_run", false, "If set, perform a dry run (no file downloads, only show statistics)")
	flag.Parse()

	// Fallback to environment variables if flags are not provided
	user := *userFlag
	if user == "" {
		user = os.Getenv("SYNOLOGY_NAS_USER")
	}

	pass := *passFlag
	if pass == "" {
		pass = os.Getenv("SYNOLOGY_NAS_PASS")
	}

	url := *urlFlag
	if url == "" {
		url = os.Getenv("SYNOLOGY_NAS_URL")
	}

	downloadDir := *downloadDirFlag
	if downloadDir == "" {
		downloadDir = os.Getenv("SYNOLOGY_DOWNLOAD_DIR")
	}
	// Use current directory if no download directory is specified
	if downloadDir == "" {
		downloadDir = "."
	}

	// Check if directory exists
	if stat, err := os.Stat(downloadDir); err != nil || !stat.IsDir() {
		if err != nil {
			fmt.Printf("Warning: Download directory '%s' does not exist. Attempting to create it.", downloadDir)
			if err := os.MkdirAll(downloadDir, 0755); err != nil {
				log.Fatalf("Failed to create download directory: %v", err)
			}
		} else {
			log.Fatalf("Error: Specified path '%s' is not a directory", downloadDir)
		}
	}

	// Convert to absolute path
	downloadDir, err := filepath.Abs(downloadDir)
	if err != nil {
		log.Fatalf("Failed to resolve absolute path of download directory: %v", err)
	}
	fmt.Printf("Files will be downloaded to: %s", downloadDir)

	if user == "" || pass == "" || url == "" {
		log.Fatalf("Missing required parameters: user, pass, and url must be provided either as flags or environment variables")
	}

	fmt.Printf("Synology Office Exporter v%s\n", Version)
	exporter, err := syndexp.NewExporter(user, pass, url, downloadDir, syndexp.WithDryRun(*dryRunFlag))
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	exitCode := 0

	sources, err := parseSources(*sourcesFlag)
	if err != nil {
		log.Fatalf("Error parsing sources: %v. Valid sources are: %s, %s, %s",
			err, sourceMyDrive, sourceTeamFolder, sourceShared)
	}

	// Run the export for each specified source
	for _, source := range sources {
		var stats syndexp.ExportStats
		var err error

		switch source {
		case sourceMyDrive:
			stats, err = exporter.ExportMyDrive()
		case sourceTeamFolder:
			stats, err = exporter.ExportTeamFolder()
		case sourceShared:
			stats, err = exporter.ExportSharedWithMe()
		default:
			continue // Shouldn't happen due to parseSources validation
		}

		if err != nil {
			exitCode = 1
			fmt.Printf("Export [%s] failed: %v\n", source, err)
			continue
		}

		fmt.Printf("[%s] Downloaded: %d, Skipped: %d, Ignored: %d, Removed:	 %d, DownloadErrs: %d, RemoveErrs: %d\n",
			source, stats.Downloaded, stats.Skipped, stats.Ignored, stats.Removed, stats.DownloadErrs, stats.RemoveErrs)

		if stats.TotalErrs() > 0 {
			exitCode = 1
		}
	}

	fmt.Println("Export complete")
	os.Exit(exitCode)
}

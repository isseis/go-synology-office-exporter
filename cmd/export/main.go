package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"github.com/isseis/go-synology-office-exporter/logger"
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

// printUsage prints the complete usage information including flags and environment variables
func printUsage() {
	// Print standard flag usage
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()

	// Print logger environment variables
	fmt.Fprintln(flag.CommandLine.Output(), "\nLogger environment variables:")
	loggerEnvVars := logger.GetEnvVarsHelp()
	for _, v := range loggerEnvVars {
		fmt.Fprintf(flag.CommandLine.Output(), "  %-20s %s\n", v.Name, v.Description)
	}

	// Print Synology NAS environment variables
	synologyEnvVars := []struct {
		Name        string
		Description string
	}{
		{"SYNOLOGY_NAS_USER", "Synology NAS username"},
		{"SYNOLOGY_NAS_PASS", "Synology NAS password"},
		{"SYNOLOGY_NAS_URL", "Synology NAS URL"},
		{"SYNOLOGY_DOWNLOAD_DIR", "Directory to save downloaded files (default: current directory)"},
	}

	fmt.Fprintln(flag.CommandLine.Output(), "\nSynology NAS environment variables:")
	for _, v := range synologyEnvVars {
		fmt.Fprintf(flag.CommandLine.Output(), "  %-20s %s\n", v.Name, v.Description)
	}
}

func main() {
	flag.Usage = printUsage

	// Define command-line flags for Synology connection (not handled by config)
	userFlag := flag.String("user", "", "Synology NAS username")
	passFlag := flag.String("pass", "", "Synology NAS password")
	urlFlag := flag.String("url", "", "Synology NAS URL")
	downloadDirFlag := flag.String("output", "", "Directory to save downloaded files")
	sourcesFlag := flag.String("sources", "mydrive,teamfolder,shared", "Comma-separated list of sources to export (mydrive,teamfolder,shared)")
	dryRunFlag := flag.Bool("dry_run", false, "If set, perform a dry run (no file downloads, only show statistics)")

	// Parse all flags
	flag.Parse()

	// Now load config which will use the parsed flag values
	cfg, err := logger.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading logger config: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	log := logger.NewHybridLogger(*cfg)
	defer func() {
		if err := log.FlushWebhook(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to flush webhook logs: %v\n", err)
		}
	}()

	fmt.Println("Starting Synology Office Exporter...")

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
	if downloadDir == "" {
		downloadDir = "."
	}
	if stat, err := os.Stat(downloadDir); err != nil || !stat.IsDir() {
		if err != nil {
			fmt.Printf("Warning: Download directory '%s' does not exist. Attempting to create it.\n", downloadDir)
			if err := os.MkdirAll(downloadDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create download directory: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: Specified path '%s' is not a directory\n", downloadDir)
			os.Exit(1)
		}
	}
	downloadDir, err = filepath.Abs(downloadDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resolve absolute path of download directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Files will be downloaded to: %s\n", downloadDir)

	if user == "" || pass == "" || url == "" {
		fmt.Fprintf(os.Stderr, "Missing required parameters: user, pass, and url must be provided either as flags or environment variables\n")
		os.Exit(1)
	}

	log.Info("Synology Office Exporter started", "version", Version)
	exporter, err := syndexp.NewExporter(user, pass, url, downloadDir, syndexp.WithDryRun(*dryRunFlag))
	if err != nil {
		log.Error("Failed to create exporter", "error", err)
		os.Exit(1)
	}

	exitCode := 0

	sources, err := parseSources(*sourcesFlag)
	if err != nil {
		log.Error("Error parsing sources", "error", err, "valid_sources", []string{string(sourceMyDrive), string(sourceTeamFolder), string(sourceShared)})
		os.Exit(1)
	}

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
			continue
		}
		if err != nil {
			exitCode = 1
			log.Error("Export failed", "source", source, "error", err)
			fmt.Printf("Export [%s] failed: %v\n", source, err)
			continue
		}
		log.Info("Export completed", "source", source, "downloaded", stats.Downloaded, "skipped", stats.Skipped, "ignored", stats.Ignored, "removed", stats.Removed, "download_errs", stats.DownloadErrs, "remove_errs", stats.RemoveErrs)
		fmt.Printf("[%s] Downloaded: %d, Skipped: %d, Ignored: %d, Removed: %d, DownloadErrs: %d, RemoveErrs: %d\n",
			source, stats.Downloaded, stats.Skipped, stats.Ignored, stats.Removed, stats.DownloadErrs, stats.RemoveErrs)
		if stats.TotalErrs() > 0 {
			exitCode = 1
		}
	}
	log.Info("Export complete")
	fmt.Println("Export complete")
	os.Exit(exitCode)
}

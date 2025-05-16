package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	syndexp "github.com/isseis/go-synology-office-exporter/synology_drive_exporter"
	"github.com/joho/godotenv"
)

const Version = "0.1.0"

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}
}

func main() {
	log.Println("Starting Synology Office Exporter...")

	// Define command-line flags
	userFlag := flag.String("user", "", "Synology NAS username")
	passFlag := flag.String("pass", "", "Synology NAS password")
	urlFlag := flag.String("url", "", "Synology NAS URL")
	downloadDirFlag := flag.String("output", "", "Directory to save downloaded files")
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
	exporter, err := syndexp.NewExporter(user, pass, url, downloadDir)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	exitCode := 0

	type exportTask struct {
		name string
		fn   func() (syndexp.ExportStats, error)
	}
	tasks := []exportTask{
		{"MyDrive", exporter.ExportMyDrive},
		{"TeamFolder", exporter.ExportTeamFolder},
		{"SharedWithMe", exporter.ExportSharedWithMe},
	}
	for _, task := range tasks {
		stats, err := task.fn()
		if err != nil {
			exitCode = 1
			fmt.Printf("Export [%s] failed: %v\n", task.name, err)
			continue
		}
		fmt.Printf("[%s] Downloaded: %d, Skipped: %d, Ignored: %d, Errors: %d\n",
			task.name, stats.Downloaded, stats.Skipped, stats.Ignored, stats.Errors)
		if stats.Errors > 0 {
			exitCode = 1
		}
	}

	fmt.Println("Export complete")
	os.Exit(exitCode)
}

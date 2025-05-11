package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_exporter"
	"github.com/joho/godotenv"
)

// Version contains the current version of the application
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

	if user == "" || pass == "" || url == "" {
		log.Fatalf("Missing required parameters: user, pass, and url must be provided either as flags or environment variables")
	}

	// Print version information
	fmt.Printf("Synology Office Exporter v%s\n", Version)
	exporter, err := synd.NewExporter(user, pass, url)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	if err := exporter.ExportMyDrive(); err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	log.Println("Export complete")
	os.Exit(0)
}

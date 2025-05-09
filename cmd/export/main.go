package main

import (
	"fmt"
	"log"
	"os"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_exporter"
)

// Version contains the current version of the application
const Version = "0.1.0"

func main() {
	log.Println("Starting Synology Office Exporter...")

	// Print version information
	fmt.Printf("Synology Office Exporter v%s\n", Version)
	exporter, err := synd.NewExporter(
		os.Getenv("SYNOLOGY_NAS_USER"),
		os.Getenv("SYNOLOGY_NAS_PASS"),
		os.Getenv("SYNOLOGY_NAS_URL"),
	)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	if err := exporter.ExportMyDrive(); err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	log.Println("Export complete")
	os.Exit(0)
}

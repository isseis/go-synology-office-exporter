package main

import (
	"fmt"
	"log"
	"os"
)

// Version contains the current version of the application
const Version = "0.1.0"

func main() {
	log.Println("Starting Synology Office Exporter...")

	// Print version information
	fmt.Printf("Synology Office Exporter v%s\n", Version)

	// TODO: Implement exporter functionality

	log.Println("Export complete")
	os.Exit(0)
}

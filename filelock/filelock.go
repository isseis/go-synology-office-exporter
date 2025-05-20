// Package filelock provides a simple file-based mutual exclusion lock.
// It ensures that only one process can hold a lock for a given file at a time,
// even across multiple processes.
package filelock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LockInfo contains information about the lock holder.
type LockInfo struct {
	PID       int    `json:"pid"`       // Process ID of the lock holder
	Timestamp string `json:"timestamp"` // When the lock was acquired (RFC3339 format)
	Hostname  string `json:"hostname"`  // Host where the lock was acquired
}

// ErrLockHeld is returned when attempting to acquire a lock that is already held.
var ErrLockHeld = fmt.Errorf("lock already held")

// TryLock attempts to acquire a lock for the given file.
// Returns a function to release the lock, or an error if the lock could not be acquired.
// The lock is automatically released if the process exits.
func TryLock(path string) (func(), error) {
	// Convert to absolute path to handle relative paths consistently
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create lock file with .lock extension
	lockFile := absPath + ".lock"

	// Check if lock file exists and read its content if it does
	if _, err := os.Stat(lockFile); err == nil {
		return nil, ErrLockHeld
	}

	// Prepare lock info
	info := LockInfo{
		PID:       os.Getpid(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	info.Hostname, _ = os.Hostname() // Ignore error, hostname is optional

	// Create and write lock file
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, ErrLockHeld
		}
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ") // Pretty print for debugging
	if err := encoder.Encode(info); err != nil {
		os.Remove(lockFile) // Clean up if we fail to write
		return nil, fmt.Errorf("failed to write lock info: %w", err)
	}

	// Ensure the data is written to disk
	if err := f.Sync(); err != nil {
		os.Remove(lockFile) // Clean up if we fail to sync
		return nil, fmt.Errorf("failed to sync lock file: %w", err)
	}

	// Create cleanup function
	unlock := func() {
		if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
			// Log the error but don't return it as the unlock function signature doesn't support it
			fmt.Fprintf(os.Stderr, "Warning: failed to remove lock file: %v\n", err)
		}
	}

	return unlock, nil
}

// ReadLockInfo reads and parses the lock file information.
// Returns the lock info if the file exists and is valid, or an error otherwise.
func ReadLockInfo(path string) (*LockInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	lockFile := absPath + ".lock"
	data, err := os.ReadFile(lockFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var info LockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}

	return &info, nil
}

// Package filelock provides a simple file-based mutual exclusion lock.
// It ensures that only one process can hold a lock for a given file at a time,
// even across multiple processes.
package filelock

import (
	"fmt"
	"os"
	"path/filepath"
)

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

	// Try to create the lock file exclusively
	// O_EXCL ensures that this call creates the file - if it already exists, it will fail
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, ErrLockHeld
		}
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}
	f.Close()

	// Create cleanup function
	unlock := func() {
		os.Remove(lockFile)
	}

	return unlock, nil
}

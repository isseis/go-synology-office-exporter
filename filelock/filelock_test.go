package filelock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTryLock(t *testing.T) {
	tempDir := t.TempDir()
	lockFile := filepath.Join(tempDir, "test.lock")

	// First lock should succeed
	unlock1, err := TryLock(lockFile)
	if err != nil {
		t.Fatalf("First TryLock failed: %v", err)
	}

	// Second lock should fail
	_, err = TryLock(lockFile)
	if err != ErrLockHeld {
		t.Errorf("Expected ErrLockHeld, got %v", err)
	}

	// Unlock first lock
	unlock1()

	// Should be able to lock again after unlock
	unlock2, err := TryLock(lockFile)
	if err != nil {
		t.Fatalf("Third TryLock failed: %v", err)
	}
	unlock2()
}

func TestLockFileCleanup(t *testing.T) {
	tempDir := t.TempDir()
	lockFile := filepath.Join(tempDir, "test.lock")

	unlock, err := TryLock(lockFile)
	if err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}

	// Lock file should exist
	if _, err := os.Stat(lockFile + ".lock"); os.IsNotExist(err) {
		t.Error("Lock file was not created")
	}

	unlock()

	// Lock file should be removed
	if _, err := os.Stat(lockFile + ".lock"); !os.IsNotExist(err) {
		t.Error("Lock file was not removed")
	}
}

func TestConcurrentLocks(t *testing.T) {
	tempDir := t.TempDir()
	lockFile := filepath.Join(tempDir, "concurrent.lock")

	// Channel to coordinate goroutines
	firstLockAcquired := make(chan struct{})
	secondTryStarted := make(chan struct{})
	errCh := make(chan error, 1)

	// First goroutine - acquires the lock
	go func() {
		unlock, err := TryLock(lockFile)
		if err != nil {
			errCh <- fmt.Errorf("first lock failed: %w", err)
			return
		}
		defer unlock()

		// Signal that first lock is acquired
		close(firstLockAcquired)

		// Wait for second goroutine to try acquiring the lock
		<-secondTryStarted
	}()

	// Second goroutine - tries to acquire the same lock
	go func() {
		// Wait for first goroutine to acquire the lock
		<-firstLockAcquired
		close(secondTryStarted)

		// Try to acquire the lock - this should fail
		_, err := TryLock(lockFile)
		errCh <- err
	}()

	// Check results
	err := <-errCh
	if err != ErrLockHeld {
		t.Errorf("Expected ErrLockHeld, got %v", err)
	}
}

func TestLockInReadOnlyDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping test when running as root user")
	}

	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a subdirectory with read-only permissions
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0o555); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	// Try to create a lock file in the read-only directory
	lockFile := filepath.Join(readOnlyDir, "test.lock")
	_, err := TryLock(lockFile)

	// We expect a permission denied error
	if err == nil {
		t.Error("Expected an error when creating lock in read-only directory")
	} else if !os.IsPermission(err) && !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Expected permission denied error, got: %v", err)
	}

	// Verify no lock file was created
	if _, err := os.Stat(lockFile + ".lock"); !os.IsNotExist(err) {
		t.Error("Lock file was created in read-only directory")
	}
}

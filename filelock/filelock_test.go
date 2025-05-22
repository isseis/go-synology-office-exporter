package filelock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	testComplete := make(chan struct{})
	errCh := make(chan error, 2) // Buffer for both possible errors

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

		// Wait for test to complete
		<-testComplete
	}()

	// Second goroutine - tries to acquire the same lock
	go func() {
		// Wait for first goroutine to acquire the lock
		<-firstLockAcquired

		// Try to acquire the lock - this should fail
		_, err := TryLock(lockFile)
		errCh <- err

		// Signal test completion
		close(testComplete)
	}()

	// Check results
	err := <-errCh
	if err != ErrLockHeld {
		t.Fatalf("Expected ErrLockHeld, got %v", err)
	}

	// Ensure we don't have any unexpected errors
	select {
	case err := <-errCh:
		t.Fatalf("Unexpected error from goroutines: %v", err)
	default:
		// No more errors, test passed
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

func TestLockFileContent(t *testing.T) {
	tempDir := t.TempDir()
	lockFile := filepath.Join(tempDir, "content_test.lock")

	// Acquire lock
	unlock, err := TryLock(lockFile)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	defer unlock()

	// Read and verify lock file content
	info, err := ReadLockInfo(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock info: %v", err)
	}

	// Verify PID
	if info.PID != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), info.PID)
	}

	// Verify timestamp is recent (within 1 second)
	ts, err := time.Parse(time.RFC3339, info.Timestamp)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}
	if time.Since(ts) > time.Second {
		t.Errorf("Timestamp %v is too old", ts)
	}

	// Verify hostname (if available)
	if hostname, _ := os.Hostname(); hostname != "" && info.Hostname != hostname {
		t.Errorf("Expected hostname %q, got %q", hostname, info.Hostname)
	}

	// Verify the file is valid JSON
	data, err := os.ReadFile(lockFile + ".lock")
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Fatalf("Lock file is not valid JSON: %v", err)
	}

	// Check required fields
	requiredFields := []string{"pid", "timestamp"}
	for _, field := range requiredFields {
		if _, exists := jsonData[field]; !exists {
			t.Errorf("Missing required field in lock file: %s", field)
		}
	}
}

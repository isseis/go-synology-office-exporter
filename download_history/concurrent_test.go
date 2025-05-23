//go:build test
// +build test

package download_history

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"time"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
	"github.com/stretchr/testify/assert"
)

// TestConcurrentReadAccess verifies that multiple goroutines can safely call read methods concurrently.
func TestConcurrentReadAccess(t *testing.T) {
	t.Parallel()
	dh := NewDownloadHistoryForTest(t, map[string]DownloadItem{})
	defer dh.Close()
	var wg sync.WaitGroup
	readers := 10
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = dh.GetStats()
			_, _, _ = dh.GetItem("item1")
		}(i)
	}
	wg.Wait()
}

// TestConcurrentWriteAccess verifies that write operations are mutually exclusive and safe under concurrency.
func TestConcurrentWriteAccess(t *testing.T) {
	t.Parallel()
	dh := NewDownloadHistoryForTest(t, map[string]DownloadItem{})
	defer dh.Close()
	var wg sync.WaitGroup
	writers := 10
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			item := DownloadItem{DownloadStatus: StatusLoaded}
			_ = dh.SetDownloaded("item"+string(rune('A'+idx)), item)
			_ = dh.MarkSkipped("item1")
		}(i)
	}
	wg.Wait()
}

// TestConcurrentReadWriteMix verifies that concurrent reads and writes do not corrupt state or panic.
func TestConcurrentReadWriteMix(t *testing.T) {
	t.Parallel()
	dh := NewDownloadHistoryForTest(t, map[string]DownloadItem{})
	defer dh.Close()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = dh.GetStats()
				_, _, _ = dh.GetItem("item1")
			}
		}()
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				item := DownloadItem{DownloadStatus: StatusLoaded}
				_ = dh.SetDownloaded("item"+string(rune('A'+idx)), item)
				_ = dh.MarkSkipped("item1")
			}
		}(i)
	}
	wg.Wait()
}

// TestAtomicCounters verifies that concurrent increments to counters are atomic and correct.
func TestAtomicCounters(t *testing.T) {
	t.Parallel()
	var c counter
	var wg sync.WaitGroup
	incr := 1000
	for i := 0; i < incr; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Increment()
		}()
	}
	wg.Wait()
	if c.Get() != incr {
		t.Errorf("expected %d, got %d", incr, c.Get())
	}
}

// TestStateMachineConcurrent verifies state transitions and constraints under concurrent access.
func TestStateMachineConcurrent(t *testing.T) {
	dh := NewDownloadHistoryForTest(t, map[string]DownloadItem{
		"item1": {DownloadStatus: StatusLoaded},
	}, WithTempDir("testhistory.json"))
	// GetStats and GetItem should succeed in stateReady
	if _, err := dh.GetStats(); err != nil {
		t.Errorf("GetStats before Save should succeed: %v", err)
	}
	if _, _, err := dh.GetItem("item1"); err != nil {
		t.Errorf("GetItem before Save should succeed: %v", err)
	}
	// Call Save to transition to stateSaved
	if err := dh.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	// After Save, only GetObsoleteItems should succeed
	if _, err := dh.GetStats(); err != ErrNotReady {
		t.Errorf("GetStats after Save should return ErrNotReady, got: %v", err)
	}
	if _, _, err := dh.GetItem("item1"); err != ErrNotReady {
		t.Errorf("GetItem after Save should return ErrNotReady, got: %v", err)
	}
	if _, err := dh.GetObsoleteItems(); err != nil {
		t.Errorf("GetObsoleteItems after Save should succeed: %v", err)
	}
}

// TestConcurrentLoad verifies that only one Load operation succeeds when called concurrently.
func TestConcurrentLoad(t *testing.T) {
	t.Parallel()
	dh := NewDownloadHistoryForTest(t, map[string]DownloadItem{}, WithInitialState(stateNew))
	defer dh.Close()

	var wg sync.WaitGroup
	const numLoaders = 5
	var loadCount int32
	var loadErrors int32

	for i := 0; i < numLoaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dh.Load(); err != nil {
				if err != ErrAlreadyLoaded {
					atomic.AddInt32(&loadErrors, 1)
				}
				return
			}
			atomic.AddInt32(&loadCount, 1)
		}()
	}
	wg.Wait()

	if loadCount != 1 {
		t.Errorf("Expected exactly 1 successful Load, got %d", loadCount)
	}
	if loadErrors > 0 {
		t.Errorf("Got %d unexpected load errors", loadErrors)
	}
}

// TestConcurrentSave verifies that only one Save operation succeeds when called concurrently.
func TestConcurrentSave(t *testing.T) {
	t.Parallel()
	dh := NewDownloadHistoryForTest(t, map[string]DownloadItem{}, WithTempDir("test-history.json"))
	assert.Equal(t, stateReady, dh.state)

	var wg sync.WaitGroup
	const numSavers = 5
	var saveCount int32
	for range numSavers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dh.Save(); err == nil {
				atomic.AddInt32(&saveCount, 1)
			}
		}()
	}
	wg.Wait()

	if saveCount != 1 {
		t.Errorf("Expected exactly 1 successful Save, got %d", saveCount)
	}
}

// TestConcurrentLoadSave verifies that Load and Save operations don't interfere with each other.
func TestConcurrentLoadSave(t *testing.T) {
	t.Parallel()
	dh := NewDownloadHistoryForTest(t, map[string]DownloadItem{}, WithTempDir("test-history.json"))
	assert.Equal(t, stateReady, dh.state)
	var wg sync.WaitGroup
	const numOps = 10
	var loadCount, saveCount int32

	// Initial load to get to ready state has already been done in NewDownloadHistoryForTest
	atomic.AddInt32(&loadCount, 1)

	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dh.Load(); err == nil {
				atomic.AddInt32(&loadCount, 1)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dh.Save(); err == nil {
				atomic.AddInt32(&saveCount, 1)
			}
		}()
	}
	wg.Wait()

	// We should have exactly 1 successful Load and 1 successful Save
	if loadCount != 1 {
		t.Errorf("Expected exactly 1 successful Load, got %d", loadCount)
	}
	if saveCount < 1 {
		t.Error("Expected at least 1 successful Save, got 0")
	}
}

// TestConcurrentGetObsoleteItems verifies that GetObsoleteItems works correctly with concurrent access.
func TestConcurrentGetObsoleteItems(t *testing.T) {
	t.Parallel()
	dh := NewDownloadHistoryForTest(t, map[string]DownloadItem{}, WithTempDir("test-history.json"))
	assert.Equal(t, stateReady, dh.state)

	// Set 10 items
	items := map[string]DownloadItem{}
	for i := range 10 {
		fileID := synd.FileID(strconv.Itoa(i))
		hash := synd.FileHash("hash" + string(rune('A'+i)))
		item := DownloadItem{
			FileID:         fileID,
			Hash:           hash,
			DownloadTime:   time.Now(),
			DownloadStatus: StatusLoaded,
		}
		items["item"+string(rune('A'+i))] = item
	}
	dh.items = items

	// Save to move to stateSaved
	if err := dh.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	const numReaders = 5
	results := make([][]string, numReaders) // Pre-allocate slice with known size
	var wg sync.WaitGroup

	for i := range numReaders {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			obsolete, err := dh.GetObsoleteItems()
			if err != nil {
				t.Errorf("GetObsoleteItems failed: %v", err)
				return
			}
			results[idx] = obsolete // Safe assignment by index
		}(i)
	}
	wg.Wait()

	// All results should be the same length (10 items)
	expectedLen := 10
	for i, res := range results {
		if len(res) != expectedLen {
			t.Errorf("Result %d: expected %d items, got %d", i, expectedLen, len(res))
		}
	}
}

// TestStateTransitions verifies state transitions work correctly under concurrent access.
func TestStateTransitions(t *testing.T) {
	t.Parallel()
	dh := NewDownloadHistoryForTest(t, map[string]DownloadItem{},
		WithTempDir("test-history.json"),
		WithInitialState(stateNew),
		WithLoadCallback(func() {
			// Wait in Load callback to simulate long-running Load
			time.Sleep(100 * time.Millisecond)
		}))
	var wg sync.WaitGroup

	// Test concurrent Load operations
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := dh.Load(); err != nil {
			t.Errorf("Load failed: %v", err)
		}
	}()

	// Wait a bit to ensure Load has started
	time.Sleep(10 * time.Millisecond)

	// Try to Save while Load is in progress
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dh.Save()
		// Save() should be blocked until Load() finishes
		assert.NoError(t, err)
	}()

	wg.Wait()

	// Verify final state is ready
	dh.mu.RLock()
	defer dh.mu.RUnlock()
	if dh.state != stateSaved {
		t.Errorf("Expected stateSaved, got %v", dh.state)
	}
}

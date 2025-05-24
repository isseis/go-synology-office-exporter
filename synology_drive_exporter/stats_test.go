package synology_drive_exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExportStats(t *testing.T) {
	t.Run("No errors", func(t *testing.T) {
		stats := &ExportStats{
			Downloaded: 5,
			Skipped:    3,
			Ignored:    2,
			Removed:    1,
		}

		assert.Equal(t, 0, stats.TotalErrs(), "TotalErrs() should return 0 when no errors")
		expectedString := "downloaded=5, skipped=3, ignored=2, removed=1, download_errors=0, remove_errors=0"
		assert.Equal(t, expectedString, stats.String(), "String() should return the expected format")
	})

	t.Run("With download errors", func(t *testing.T) {
		stats := &ExportStats{
			Downloaded:   2,
			DownloadErrs: 1,
		}

		assert.Equal(t, 1, stats.TotalErrs(), "TotalErrs() should include download errors")
		expectedString := "downloaded=2, skipped=0, ignored=0, removed=0, download_errors=1, remove_errors=0"
		assert.Equal(t, expectedString, stats.String(), "String() should include download errors")
	})

	t.Run("With remove errors", func(t *testing.T) {
		stats := &ExportStats{
			Removed:    2,
			RemoveErrs: 1,
		}

		assert.Equal(t, 1, stats.TotalErrs(), "TotalErrs() should include remove errors")
		expectedString := "downloaded=0, skipped=0, ignored=0, removed=2, download_errors=0, remove_errors=1"
		assert.Equal(t, expectedString, stats.String(), "String() should include remove errors")
	})

	t.Run("Increment methods", func(t *testing.T) {
		t.Run("IncrementDownloadErrs", func(t *testing.T) {
			stats := &ExportStats{}
			initialErrs := stats.TotalErrs()
			stats.IncrementDownloadErrs()
			assert.Equal(t, initialErrs+1, stats.TotalErrs(), "Should increment total errors by 1")
			assert.Equal(t, 1, stats.DownloadErrs, "Should increment DownloadErrs counter")
		})

		t.Run("IncrementRemoveErrs", func(t *testing.T) {
			stats := &ExportStats{}
			initialErrs := stats.TotalErrs()
			stats.IncrementRemoveErrs()
			assert.Equal(t, initialErrs+1, stats.TotalErrs(), "Should increment total errors by 1")
			assert.Equal(t, 1, stats.RemoveErrs, "Should increment RemoveErrs counter")
		})

		t.Run("IncrementRemoved", func(t *testing.T) {
			stats := &ExportStats{}
			stats.IncrementRemoved()
			assert.Equal(t, 1, stats.Removed, "Should increment Removed counter")
		})
	})
}

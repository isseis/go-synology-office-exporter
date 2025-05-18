package download_history

// NewDownloadHistoryForTest creates a DownloadHistory instance initialized with the given items map.
// This is intended for test code convenience.
//
// Note: This function is included in all builds unless this file is restricted to test builds via a build tag.
func NewDownloadHistoryForTest(items map[string]DownloadItem) *DownloadHistory {
	dh := &DownloadHistory{}
	dh.items = items
	return dh
}

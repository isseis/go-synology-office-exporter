package synology_drive_api

// SynologyClient is an interface for Synology Drive API client operations.
type SynologyClient interface {
	Login() error
	Logout() error
	// Add other methods as needed for testing
}

// NewClientFactory returns a SynologyClient.
// If useMock is true, returns a mock client; otherwise, returns a real client.
func NewClientFactory(user, pass, url string, useMock bool) SynologyClient {
	if useMock {
		return NewMockSynologyClient()
	}
	s, _ := NewSynologySession(user, pass, url)
	return s
}

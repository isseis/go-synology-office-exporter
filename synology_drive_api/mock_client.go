package synology_drive_api

import "errors"

// MockSynologyClient is a mock implementation of SynologyClient for testing.
type MockSynologyClient struct {
	LoggedIn   bool
	FailLogin  bool
	FailLogout bool
}

func NewMockSynologyClient() *MockSynologyClient {
	return &MockSynologyClient{}
}

// Login simulates a login operation.
func (m *MockSynologyClient) Login() error {
	if m.FailLogin {
		return errors.New("mock: login failed")
	}
	m.LoggedIn = true
	return nil
}

// Logout simulates a logout operation.
func (m *MockSynologyClient) Logout() error {
	if m.FailLogout {
		return errors.New("mock: logout failed")
	}
	m.LoggedIn = false
	return nil
}

package synology_drive_api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuth tests the login and logout functionality of SynologyClient.
// By default, tests use the mock client for SynologyClient.
// To run tests against a real NAS, set USE_REAL_SYNOLOGY_API=1 in your environment.
func TestAuth(t *testing.T) {
	useReal := os.Getenv("USE_REAL_SYNOLOGY_API") == "1"
	useMock := !useReal
	user := getNasUser()
	pass := getNasPass()
	url := getNasUrl()
	client := NewClientFactory(user, pass, url, useMock)

	t.Run("Login", func(t *testing.T) {
		err := client.Login()
		require.NoError(t, err)
		if useMock {
			mockClient := client.(*MockSynologyClient)
			assert.True(t, mockClient.LoggedIn)
		} else {
			s := client.(*SynologySession)
			assert.False(t, s.sessionExpired())
			assert.NotEmpty(t, s.sid)
		}
	})

	t.Run("Logout", func(t *testing.T) {
		err := client.Login()
		require.NoError(t, err)
		err = client.Logout()
		require.NoError(t, err)
		if useMock {
			mockClient := client.(*MockSynologyClient)
			assert.False(t, mockClient.LoggedIn)
		} else {
			s := client.(*SynologySession)
			assert.True(t, s.sessionExpired())
			assert.Empty(t, s.sid)
		}
	})
}

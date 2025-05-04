package synology_drive_api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpGet(t *testing.T) {
	s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
	url := s.buildUrl("auth.cgi", map[string]string{
		"api":     "SYNO.API.Auth",
		"method":  "login",
		"version": "3",
		"account": getNasUser(),
		"passwd":  getNasPass(),
		"session": "SynologyDrive",
		"format":  "cookie",
	})
	require.Nil(t, err)

	// Verify the structure of the URL but exclude the password
	expectedUrl := getNasUrl() + "/webapi/auth.cgi"
	assert.Contains(t, url.String(), expectedUrl)
	assert.Contains(t, url.String(), "account="+getNasUser())
	assert.Contains(t, url.String(), "api=SYNO.API.Auth")
	assert.Contains(t, url.String(), "format=cookie")
	assert.Contains(t, url.String(), "method=login")
	assert.Contains(t, url.String(), "session=SynologyDrive")
	assert.Contains(t, url.String(), "version=3")
	// Password is included in the URL but not verified in the test
}

func TestNewSynologySession(t *testing.T) {
	// Create a session with a valid URL
	session, err := NewSynologySession("testuser", "testpass", "https://test.synology.com")
	require.NoError(t, err, "NewSynologySession should not fail with valid URL")
	assert.Equal(t, "testuser", session.username, "Username should match")
	assert.Equal(t, "testpass", session.password, "Password should match")
	assert.Equal(t, "test.synology.com", session.hostname, "Hostname should match")
	assert.Equal(t, "https", session.scheme, "Scheme should match")

	// Create a session with an invalid URL
	_, err = NewSynologySession("testuser", "testpass", ":invalid-url")
	assert.Error(t, err, "NewSynologySession should fail with invalid URL")
	assert.IsType(t, InvalidUrlError(""), err, "Error should be of type InvalidUrlError")
}

func TestSessionExpired(t *testing.T) {
	session, err := NewSynologySession("testuser", "testpass", "https://test.synology.com")
	require.NoError(t, err, "Failed to create test session")

	// Initially the session has no sid and is considered expired
	assert.True(t, session.sessionExpired(), "New session should be expired")

	// When sid is set, the session is not expired
	session.sid = "test-sid"
	assert.False(t, session.sessionExpired(), "Session with sid should not be expired")
}

func TestBuildUrl(t *testing.T) {
	session, err := NewSynologySession("testuser", "testpass", "https://test.synology.com")
	require.NoError(t, err, "Failed to create test session")

	// URL without parameters
	url := session.buildUrl("test.cgi", nil)
	expected := "https://test.synology.com/webapi/test.cgi"
	assert.Equal(t, expected, url.String(), "URL without parameters should match expected format")

	// URL with parameters
	params := map[string]string{
		"param1": "value1",
		"param2": "value2",
	}
	url = session.buildUrl("test.cgi", params)
	assert.Equal(t, "value1", url.Query().Get("param1"), "URL should contain correct param1 value")
	assert.Equal(t, "value2", url.Query().Get("param2"), "URL should contain correct param2 value")
}

package synology_drive_api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getEnvOrPanic(key string) string {
	if value, exists := os.LookupEnv(key); !exists {
		panic(key + " is not set")
	} else {
		return value
	}
}

func getNasUrl() string {
	return getEnvOrPanic("SYNOLOGY_NAS_URL")
}

func getNasUser() string {
	return getEnvOrPanic("SYNOLOGY_NAS_USER")
}

func getNasPass() string {
	return getEnvOrPanic("SYNOLOGY_NAS_PASS")
}

func Test1(t *testing.T) {
	t.Run("Init", func(t *testing.T) {
		s, err := NewSynologySession("username", "password", "https://example.com")
		require.Nil(t, err)
		assert.NotNil(t, s.hostname)
	})

	t.Run("httpGet", func(t *testing.T) {
		// TODO: Encrypt the password
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
	})

	t.Run("Login", func(t *testing.T) {
		s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
		require.Nil(t, err)
		err = s.Login("SynologyDrive")
		require.Nil(t, err)
	})
}

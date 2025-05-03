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

	t.Run("Login", func(t *testing.T) {
		s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
		require.Nil(t, err)
		err = s.Login()
		require.Nil(t, err)
		assert.False(t, s.sessionExpired())
		assert.NotEmpty(t, s.sid)
	})

	t.Run("Logout", func(t *testing.T) {
		s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
		require.Nil(t, err)
		err = s.Login()
		require.Nil(t, err)
		err = s.Logout()
		require.Nil(t, err)
		assert.True(t, s.sessionExpired())
		assert.Empty(t, s.sid)
	})

	t.Run("List", func(t *testing.T) {
		s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
		require.Nil(t, err)
		err = s.Login()
		require.Nil(t, err)
		resp, err := s.List(MyDrive)
		require.Nil(t, err)
		assert.Equal(t, int64(1), resp.Total)
		t.Log(resp.Items)
	})
}

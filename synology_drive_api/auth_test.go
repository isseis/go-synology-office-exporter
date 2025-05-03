package synology_drive_api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
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
}

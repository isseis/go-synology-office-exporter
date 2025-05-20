package synology_drive_api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTeamFolder(t *testing.T) {
	ResetMockLogin()
	s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
	require.NoError(t, err)
	err = s.Login()
	require.NoError(t, err)

	resp, err := s.TeamFolder(0, DefaultMaxPageSize)
	require.NoError(t, err)
	//t.Log("Response:", string(resp.raw))
	for _, item := range resp.Items {
		require.NotEmpty(t, item.FileID)
		require.NotEmpty(t, item.Name)
		require.NotEmpty(t, item.TeamID)
	}
}

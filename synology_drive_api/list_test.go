package synology_drive_api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	ResetMockLogin()
	s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
	require.NoError(t, err)
	err = s.Login()
	require.NoError(t, err)
	resp, err := s.List(MyDrive)
	require.NoError(t, err)

	// t.Log("Response:", string(resp.raw))
	for _, item := range resp.Items {
		assert.NotEmpty(t, item.FileID)
		assert.NotEmpty(t, item.Name)
		assert.NotEmpty(t, item.Path)
	}
}

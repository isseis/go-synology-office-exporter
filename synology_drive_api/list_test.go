package synology_drive_api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
	require.Nil(t, err)
	err = s.Login()
	require.Nil(t, err)
	resp, err := s.List(MyDrive)
	require.Nil(t, err)
	assert.Equal(t, int64(1), resp.Total)
	t.Log(resp.Items)
}

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

	t.Log("Response:", string(resp.raw))
	for _, item := range resp.Items {
		t.Log(item.FileID, item.DisplayPath, item.Name, item.Path, item.ContentType)
		assert.NotEmpty(t, item.FileID)
		assert.NotEmpty(t, item.Name)
		assert.NotEmpty(t, item.Path)
	}
}

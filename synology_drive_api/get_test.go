package synology_drive_api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	ResetMockLogin()
	s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
	require.NoError(t, err)

	// Test fails since the session is not logged in
	_, err = s.Get("882614125167948399")
	require.Error(t, err)

	// Test succeeds after logging in
	err = s.Login()
	require.NoError(t, err)
	resp, err := s.Get("882614125167948399")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.raw)
}

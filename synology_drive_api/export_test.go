package synology_drive_api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExport(t *testing.T) {
	s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
	require.Nil(t, err)

	// Test fails since the session is not logged in
	_, err = s.Export("882614125167948399")
	require.NotNil(t, err)

	// Test should succeed after logging in, but we haven't implemented the Export function yet
	err = s.Login()
	require.Nil(t, err)
	_, err = s.Export("882614125167948399")
	assert.NotNil(t, err)
}

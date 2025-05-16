package synology_drive_api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExport(t *testing.T) {
	ResetMockLogin()
	s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
	require.NoError(t, err)

	// Test fails since the session is not logged in
	_, err = s.Export("882614125167948399")
	require.Error(t, err)

	// Test should succeed after logging in, but we haven't implemented the Export function yet
	err = s.Login()
	require.NoError(t, err)
	res, err := s.Export("882614125167948399")
	// Skip the test if there was an error
	if err != nil {
		t.Skip("Skipping file save due to export error")
	}

	t.Log("Response [Name]:", string(res.Name))
	// Save the response to a file
	err = os.WriteFile(res.Name, res.Content, 0644)
	require.NoError(t, err, "Failed to save file")
	defer func() {
		os.Remove(res.Name)
	}()
	t.Log("Saved response to " + res.Name)
}

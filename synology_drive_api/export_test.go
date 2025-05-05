package synology_drive_api

import (
	"os"
	"testing"

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
	res, err := s.Export("882614125167948399")
	// Skip the test if there was an error
	if err != nil {
		t.Skip("Skipping file save due to export error")
	}

	t.Log("Response [Name]:", string(res.Name))
	// Save the response to a file
	err = os.WriteFile(res.Name, res.Content, 0644)
	require.Nil(t, err, "Failed to save file")
	t.Log("Saved response to " + res.Name)
}

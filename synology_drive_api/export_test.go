package synology_drive_api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestExport tests file export using SynologyClient interface.
// By default, tests use the mock client. Set USE_REAL_SYNOLOGY_API=1 to use a real NAS.
func TestExport(t *testing.T) {
	useReal := os.Getenv("USE_REAL_SYNOLOGY_API") == "1"
	useMock := !useReal
	user := getNasUser()
	pass := getNasPass()
	url := getNasUrl()
	client := NewClientFactory(user, pass, url, useMock)

	err := client.Login()
	require.NoError(t, err)

	type exportable interface {
		Export(fileID string) (*ExportResponse, error)
	}
	expClient, ok := client.(exportable)
	if !ok {
		t.Skip("Export not implemented for this client")
	}
	res, err := expClient.Export("882614125167948399")
	if useMock {
		t.Log("Mock export result:", res)
		return
	}
	if err != nil {
		t.Skip("Skipping file save due to export error")
	}
	t.Log("Response [Name]:", string(res.Name))
	err = os.WriteFile(res.Name, res.Content, 0644)
	require.NoError(t, err, "Failed to save file")
	defer func() {
		os.Remove(res.Name)
	}()
	t.Log("Saved response to " + res.Name)
}

package synology_drive_api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTeamFolder tests team folder listing using SynologyClient interface.
// By default, tests use the mock client. Set USE_REAL_SYNOLOGY_API=1 to use a real NAS.
func TestTeamFolder(t *testing.T) {
	useReal := os.Getenv("USE_REAL_SYNOLOGY_API") == "1"
	useMock := !useReal
	user := getNasUser()
	pass := getNasPass()
	url := getNasUrl()
	client := NewClientFactory(user, pass, url, useMock)

	err := client.Login()
	require.NoError(t, err)

	type teamFolderable interface {
		TeamFolder() (*TeamFolderResponse, error)
	}
	tfClient, ok := client.(teamFolderable)
	if !ok {
		t.Skip("TeamFolder not implemented for this client")
	}
	resp, err := tfClient.TeamFolder()
	if useMock {
		t.Log("Mock team folder result:", resp)
		return
	}
	require.NoError(t, err)
	for _, item := range resp.Items {
		require.NotEmpty(t, item.FileID)
		require.NotEmpty(t, item.Name)
		require.NotEmpty(t, item.TeamID)
	}
	/* Sample Response:
	{
		"data": {
			"items": [
				{
					"capabilities": {
						"can_comment": false,
						"can_delete": false,
						"can_download": true,
						"can_encrypt": false,
						"can_organize": false,
						"can_preview": true,
						"can_read": true,
						"can_rename": false,
						"can_share": false,
						"can_sync": true,
						"can_write": false
					},
					"disable_download": false,
					"enable_versioning": true,
					"file_id": "871903949079224322",
					"keep_versions": 8,
					"name": "Family",
					"team_id": "2"
				}
			],
			"total": 1
		},
		"success": true
	}
	*/
}

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

	resp, err := s.TeamFolder()
	require.NoError(t, err)
	//t.Log("Response:", string(resp.raw))
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

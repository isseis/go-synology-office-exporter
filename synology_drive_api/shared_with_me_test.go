package synology_drive_api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSharedWithMe tests shared-with-me listing using SynologyClient interface.
// By default, tests use the mock client. Set USE_REAL_SYNOLOGY_API=1 to use a real NAS.
func TestSharedWithMe(t *testing.T) {
	useReal := os.Getenv("USE_REAL_SYNOLOGY_API") == "1"
	useMock := !useReal
	user := getNasUser()
	pass := getNasPass()
	url := getNasUrl()
	client := NewClientFactory(user, pass, url, useMock)

	err := client.Login()
	require.NoError(t, err)

	type sharedWithMeable interface {
		SharedWithMe() (*SharedWithMeResponse, error)
	}
	shClient, ok := client.(sharedWithMeable)
	if !ok {
		t.Skip("SharedWithMe not implemented for this client")
	}
	resp, err := shClient.SharedWithMe()
	if useMock {
		t.Log("Mock shared-with-me result:", resp)
		return
	}
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Total, int64(0))
	for _, item := range resp.Items {
		assert.NotEmpty(t, item.FileID)
		assert.NotEmpty(t, item.Name)
		assert.NotEmpty(t, item.Path)
	}
	/* Sample Response:
	{
		"data": {
			"items": [
				{
					"access_time": 1745194614,
					"adv_shared": false,
					"app_properties": {
						"type": "none"
					},
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
					"change_id": 11451,
					"change_time": 1743745724,
					"content_snippet": "",
					"content_type": "dir",
					"created_time": 1741145838,
					"disable_download": false,
					"display_path": "/shared-with-me/Documents",
					"dsm_path": "",
					"enable_watermark": false,
					"encrypted": false,
					"file_id": "871932547865555615",
					"force_watermark_download": false,
					"hash": "",
					"image_metadata": {
						"time": 1743745724
					},
					"labels": [],
					"max_id": 11451,
					"modified_time": 1743745724,
					"name": "Documents",
					"owner": {
						"display_name": "issei",
						"name": "issei",
						"nickname": "",
						"uid": 1026
					},
					"parent_id": "871904062772129812",
					"path": "/Documents",
					"permanent_link": "12PSXT5RcnIvv61MzgM7OAFmIqWK0CrA",
					"properties": {},
					"removed": false,
					"revisions": 3,
					"shared": true,
					"shared_with": [
						{
							"display_name": "backup",
							"inherited": false,
							"name": "backup",
							"nickname": "",
							"permission_id": "",
							"role": "viewer",
							"type": "user"
						}
					],
					"size": 0,
					"starred": false,
					"support_remote": false,
					"sync_id": 926,
					"sync_to_device": false,
					"transient": false,
					"type": "dir",
					"version_id": "926",
					"watermark_version": 0
				}
			],
			"total": 1
		},
		"success": true
	}
	*/
}

package synology_drive_api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
	require.Nil(t, err)

	// Test fails since the session is not logged in
	_, err = s.Get("882614125167948399")
	require.NotNil(t, err)

	// Test succeeds after logging in
	err = s.Login()
	require.Nil(t, err)
	resp, err := s.Get("882614125167948399")
	require.Nil(t, err)
	assert.NotEmpty(t, resp.raw)

	//t.Log("Response:", string(resp.raw))
	/* Sample Response:
	{
		"data": {
			"access_time": 1746357311,
			"adv_shared": false,
			"app_properties": {"type": "none"},
			"capabilities": {
				"can_comment": true,
				"can_delete": true,
				"can_download": true,
				"can_encrypt": true,
				"can_organize": true,
				"can_preview": true,
				"can_read": true,
				"can_rename": true,
				"can_share": true,
				"can_sync": true,
				"can_write": true
			},
			"change_id": 17,
			"change_time": 1746239530,
			"content_snippet": "",
			"content_type": "document",
			"created_time": 1746239207,
			"disable_download": false,
			"display_path": "/mydrive/planning.osheet",
			"dsm_path": "",
			"enable_watermark": false,
			"encrypted": false,
			"file_id": "882614125167948399",
			"force_watermark_download": false,
			"hash": "7dd71dd1192ca985884d934470fb9d3c",
			"image_metadata": {"time": 1746239207},
			"labels": [],
			"max_id": 17,
			"modified_time": 1746239207,
			"name": "planning.osheet",
			"owner": {
				"display_name": "backup",
				"name": "backup",
				"nickname": "",
				"uid": 1029
			},
			"parent_id": "880423525918227781",
			"path": "/planning.osheet",
			"permanent_link": "13CNgcuVDEENCkSfDkcKpr39pcK6hhPD",
			"properties": {"object_id": "1029_ELSD3CRAC557FDOVULJJJDKN50.sh"},
			"removed": false,
			"revisions": 3,
			"shared": false,
			"shared_with": [],
			"size": 720,
			"starred": false,
			"support_remote": false,
			"sync_id": 17,
			"sync_to_device": false,
			"transient": false,
			"type": "file",
			"version_id": "17",
			"watermark_version": 0
		},
		"success": true
	}
	*/
}

package synology_drive_api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	ResetMockLogin()
	s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
	require.NoError(t, err)
	err = s.Login()
	require.NoError(t, err)
	resp, err := s.List(MyDrive)
	require.NoError(t, err)

	// t.Log("Response:", string(resp.raw))
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
	        "access_time": 1746239241,
	        "adv_shared": false,
	        "app_properties": {
	          "type": "none"
	        },
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
	        "change_time": 1746239241,
	        "content_snippet": "",
	        "content_type": "dir",
	        "created_time": 1746239202,
	        "disable_download": false,
	        "display_path": "/mydrive/2025-04",
	        "dsm_path": "",
	        "enable_watermark": false,
	        "encrypted": false,
	        "file_id": "882614106016756317",
	        "force_watermark_download": false,
	        "hash": "",
	        "image_metadata": {
	          "time": 1746239241
	        },
	        "labels": [],
	        "max_id": 14,
	        "modified_time": 1746239241,
	        "name": "2025-04",
	        "owner": {
	          "display_name": "backup",
	          "name": "backup",
	          "nickname": "",
	          "uid": 1029
	        },
	        "parent_id": "880423525918227781",
	        "path": "/2025-04",
	        "permanent_link": "13CNgI0QlJMjbqcZf5k3HRoZsbCAqejY",
	        "properties": {},
	        "removed": false,
	        "revisions": 1,
	        "shared": false,
	        "shared_with": [],
	        "size": 0,
	        "starred": false,
	        "support_remote": false,
	        "sync_id": 8,
	        "sync_to_device": false,
	        "transient": false,
	        "type": "dir",
	        "version_id": "8",
	        "watermark_version": 0
	      },
	      {
	        "access_time": 1746239530,
	        "adv_shared": false,
	        "app_properties": {
	          "type": "none"
	        },
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
	        "image_metadata": {
	          "time": 1746239207
	        },
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
	        "properties": {
	          "object_id": "1029_ELSD3CRAC557FDOVULJJJDKN50.sh"
	        },
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
	      // ...
	    ],
	    "total": 6
	  },
	  "success": true
	}
	*/
}

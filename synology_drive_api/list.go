package synology_drive_api

import (
	"encoding/json"
	"fmt"
	"io"
)

// listResponseV2 represents a file or folder item in a Synology Drive listing
type ListResponseItemV2 struct {
	AccessTime    int64 `json:"access_time"`
	AdvShared     bool  `json:"adv_shared"`
	AppProperties struct {
		Type string `json:"type"`
	} `json:"app_properties"`
	Capabilities struct {
		CanComment  bool `json:"can_comment"`
		CanDelete   bool `json:"can_delete"`
		CanDownload bool `json:"can_download"`
		CanEncrypt  bool `json:"can_encrypt"`
		CanOrganize bool `json:"can_organize"`
		CanPreview  bool `json:"can_preview"`
		CanRead     bool `json:"can_read"`
		CanRename   bool `json:"can_rename"`
		CanShare    bool `json:"can_share"`
		CanSync     bool `json:"can_sync"`
		CanWrite    bool `json:"can_write"`
	} `json:"capabilities"`
	ChangeID               int64               `json:"change_id"`
	ChangeTime             int64               `json:"change_time"`
	ContentSnippet         string              `json:"content_snippet"`
	ContentType            contentType         `json:"content_type"`
	CreatedTime            int64               `json:"created_time"`
	DisableDownload        bool                `json:"disable_download"`
	DisplayPath            string              `json:"display_path"`
	DsmPath                string              `json:"dsm_path"`
	EnableWatermark        bool                `json:"enable_watermark"`
	Encrypted              bool                `json:"encrypted"`
	FileID                 SynologyDriveFileID `json:"file_id"`
	ForceWatermarkDownload bool                `json:"force_watermark_download"`
	Hash                   string              `json:"hash"`
	ImageMetadata          struct {
		Time int64 `json:"time"`
	} `json:"image_metadata"`
	Labels       []string `json:"labels"`
	MaxID        int64    `json:"max_id"`
	ModifiedTime int64    `json:"modified_time"`
	Name         string   `json:"name"`
	Owner        struct {
		DisplayName string `json:"display_name"`
		Name        string `json:"name"`
		Nickname    string `json:"nickname"`
		UID         int    `json:"uid"`
	} `json:"owner"`
	ParentID      string `json:"parent_id"`
	Path          string `json:"path"`
	PermanentLink string `json:"permanent_link"`
	Properties    struct {
		ObjectID string `json:"object_id"`
	} `json:"properties"`
	Removed          bool   `json:"removed"`
	Revisions        int    `json:"revisions"`
	Shared           bool   `json:"shared"`
	SharedWith       []any  `json:"shared_with"`
	Size             int64  `json:"size"`
	Starred          bool   `json:"starred"`
	SupportRemote    bool   `json:"support_remote"`
	SyncID           int64  `json:"sync_id"`
	SyncToDevice     bool   `json:"sync_to_device"`
	Transient        bool   `json:"transient"`
	Type             string `json:"type"`
	VersionID        string `json:"version_id"`
	WatermarkVersion int    `json:"watermark_version"`
}

type ListResponseDataV2 struct {
	Items []ListResponseItemV2 `json:"items"`
	Total int                  `json:"total"`
}

type ListResponseV2 struct {
	synologyAPIResponse
	Data ListResponseDataV2 `json:"data"`
}

// List retrieves the contents of a folder on Synology Drive.
// Parameters:
//   - file_id: The identifier of the folder to list (e.g., MyDrive constant for the root folder)
//
// Returns:
//   - *ListResponseDataV2: Data structure containing the list of items and total count
//   - error: HttpError if there was a network or request error
//   - error: SynologyError if the listing failed or the response was invalid
func (s *SynologySession) List(fileID SynologyDriveFileID) (*ListResponseDataV2, error) {
	endpoint := "entry.cgi"
	params := map[string]string{
		"api":            "SYNO.SynologyDrive.Files",
		"method":         "list",
		"version":        "2",
		"filter":         "{}",
		"sort_direction": "asc",
		"sort_by":        "owner",
		"offset":         "0",
		"limit":          "1000",
		"path":           string(fileID),
	}

	rawResp, err := s.httpGet(endpoint, params)
	if err != nil {
		return nil, err
	}
	defer rawResp.Body.Close()

	body, err := io.ReadAll(rawResp.Body)
	if err != nil {
		return nil, HttpError(err.Error())
	}

	var resp ListResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, SynologyError(err.Error())
	}
	if !resp.Success {
		return nil, SynologyError(fmt.Sprintf("List folder failed: [code=%d]", resp.Err.Code))
	}
	for i := range resp.Data.Items {
		item := resp.Data.Items[i]
		if !item.ContentType.isValid() {
			return nil, SynologyError(fmt.Sprintf("Invalid content type: %s", item.ContentType))
		}
	}
	return &resp.Data, nil
}

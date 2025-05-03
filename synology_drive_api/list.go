package synology_drive_api

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// jsonSharedWithItem represents a user or group that a file or folder is shared with
type jsonSharedWithItem struct {
	DisplayName  string       `json:"display_name"`
	Inherited    bool         `json:"inherited"`
	Name         string       `json:"name"`
	Nickname     string       `json:"nickname"`
	PermissionID string       `json:"permission_id"`
	Role         Role         `json:"role"`
	Type         SharedTarget `json:"type"`
}

// listResponseV2 represents a file or folder item in a Synology Drive listing
type jsonListResponseItemV2 struct {
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
	ChangeID               int64       `json:"change_id"`
	ChangeTime             int64       `json:"change_time"`
	ContentSnippet         string      `json:"content_snippet"`
	ContentType            contentType `json:"content_type"`
	CreatedTime            int64       `json:"created_time"`
	DisableDownload        bool        `json:"disable_download"`
	DisplayPath            string      `json:"display_path"`
	DsmPath                string      `json:"dsm_path"`
	EnableWatermark        bool        `json:"enable_watermark"`
	Encrypted              bool        `json:"encrypted"`
	FileID                 FileID      `json:"file_id"`
	ForceWatermarkDownload bool        `json:"force_watermark_download"`
	Hash                   string      `json:"hash"`
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
		UID         UserID `json:"uid"`
	} `json:"owner"`
	ParentID      FileID `json:"parent_id"`
	Path          string `json:"path"`
	PermanentLink string `json:"permanent_link"`
	Properties    struct {
		ObjectID string `json:"object_id"`
	} `json:"properties"`
	Removed          bool                 `json:"removed"`
	Revisions        int                  `json:"revisions"`
	Shared           bool                 `json:"shared"`
	SharedWith       []jsonSharedWithItem `json:"shared_with"`
	Size             int64                `json:"size"`
	Starred          bool                 `json:"starred"`
	SupportRemote    bool                 `json:"support_remote"`
	SyncID           int64                `json:"sync_id"`
	SyncToDevice     bool                 `json:"sync_to_device"`
	Transient        bool                 `json:"transient"`
	Type             Type                 `json:"type"`
	VersionID        string               `json:"version_id"`
	WatermarkVersion int                  `json:"watermark_version"`
}

type jsonListResponseDataV2 struct {
	Items []jsonListResponseItemV2 `json:"items"`
	Total int64                    `json:"total"`
}

type jsonListResponseV2 struct {
	synologyAPIResponse
	Data jsonListResponseDataV2 `json:"data"`
}

type SharedWithItem struct {
	DisplayName  string
	Inherited    bool
	Name         string
	Nickname     string
	PermissionID string
	Role         Role
	Type         SharedTarget
}

// ListResponseItem represents a file or folder item in a Synology Drive listing with proper Go types
type ListResponseItem struct {
	AccessTime    time.Time
	AdvShared     bool
	AppProperties struct {
		Type string
	}
	Capabilities struct {
		CanComment  bool
		CanDelete   bool
		CanDownload bool
		CanEncrypt  bool
		CanOrganize bool
		CanPreview  bool
		CanRead     bool
		CanRename   bool
		CanShare    bool
		CanSync     bool
		CanWrite    bool
	}
	ChangeID               int64
	ChangeTime             time.Time
	ContentSnippet         string
	ContentType            contentType
	CreatedTime            time.Time
	DisableDownload        bool
	DisplayPath            string
	DsmPath                string
	EnableWatermark        bool
	Encrypted              bool
	FileID                 FileID
	ForceWatermarkDownload bool
	Hash                   string
	ImageMetadata          struct {
		Time time.Time
	}
	Labels       []string
	MaxID        int64
	ModifiedTime time.Time
	Name         string
	Owner        struct {
		DisplayName string
		Name        string
		Nickname    string
		UID         UserID
	}
	ParentID      FileID
	Path          string
	PermanentLink string
	Properties    struct {
		ObjectID string
	}
	Removed          bool
	Revisions        int
	Shared           bool
	SharedWith       []SharedWithItem
	Size             int64
	Starred          bool
	SupportRemote    bool
	SyncID           int64
	SyncToDevice     bool
	Transient        bool
	Type             Type
	VersionID        string
	WatermarkVersion int
}

// convertSharedWithItems converts a slice of jsonSharedWithItem to a slice of SharedWithItem
func convertSharedWithItems(items []jsonSharedWithItem) []SharedWithItem {
	result := make([]SharedWithItem, len(items))
	for i, item := range items {
		result[i] = SharedWithItem(item)
	}
	return result
}

// toListResponseItem converts the JSON representation to the Go friendly representation
func (j jsonListResponseItemV2) toListResponseItem() ListResponseItem {
	return ListResponseItem{
		// Convert Unix timestamp (seconds since epoch) to time.Time
		AccessTime: time.Unix(j.AccessTime, 0),
		AdvShared:  j.AdvShared,
		AppProperties: struct {
			Type string
		}{
			Type: j.AppProperties.Type,
		},
		Capabilities: struct {
			CanComment  bool
			CanDelete   bool
			CanDownload bool
			CanEncrypt  bool
			CanOrganize bool
			CanPreview  bool
			CanRead     bool
			CanRename   bool
			CanShare    bool
			CanSync     bool
			CanWrite    bool
		}{
			CanComment:  j.Capabilities.CanComment,
			CanDelete:   j.Capabilities.CanDelete,
			CanDownload: j.Capabilities.CanDownload,
			CanEncrypt:  j.Capabilities.CanEncrypt,
			CanOrganize: j.Capabilities.CanOrganize,
			CanPreview:  j.Capabilities.CanPreview,
			CanRead:     j.Capabilities.CanRead,
			CanRename:   j.Capabilities.CanRename,
			CanShare:    j.Capabilities.CanShare,
			CanSync:     j.Capabilities.CanSync,
			CanWrite:    j.Capabilities.CanWrite,
		},
		ChangeID:               j.ChangeID,
		ChangeTime:             time.Unix(j.ChangeTime, 0),
		ContentSnippet:         j.ContentSnippet,
		ContentType:            j.ContentType,
		CreatedTime:            time.Unix(j.CreatedTime, 0),
		DisableDownload:        j.DisableDownload,
		DisplayPath:            j.DisplayPath,
		DsmPath:                j.DsmPath,
		EnableWatermark:        j.EnableWatermark,
		Encrypted:              j.Encrypted,
		FileID:                 j.FileID,
		ForceWatermarkDownload: j.ForceWatermarkDownload,
		Hash:                   j.Hash,
		ImageMetadata: struct {
			Time time.Time
		}{
			Time: time.Unix(j.ImageMetadata.Time, 0),
		},
		Labels:       j.Labels,
		MaxID:        j.MaxID,
		ModifiedTime: time.Unix(j.ModifiedTime, 0),
		Name:         j.Name,
		Owner: struct {
			DisplayName string
			Name        string
			Nickname    string
			UID         UserID
		}{
			DisplayName: j.Owner.DisplayName,
			Name:        j.Owner.Name,
			Nickname:    j.Owner.Nickname,
			UID:         j.Owner.UID,
		},
		ParentID:      j.ParentID,
		Path:          j.Path,
		PermanentLink: j.PermanentLink,
		Properties: struct {
			ObjectID string
		}{
			ObjectID: j.Properties.ObjectID,
		},
		Removed:          j.Removed,
		Revisions:        j.Revisions,
		Shared:           j.Shared,
		SharedWith:       convertSharedWithItems(j.SharedWith),
		Size:             j.Size,
		Starred:          j.Starred,
		SupportRemote:    j.SupportRemote,
		SyncID:           j.SyncID,
		SyncToDevice:     j.SyncToDevice,
		Transient:        j.Transient,
		Type:             j.Type,
		VersionID:        j.VersionID,
		WatermarkVersion: j.WatermarkVersion,
	}
}

type ListResponse struct {
	Items []ListResponseItem
	Total int64
	raw   []byte
}

// List retrieves the contents of a folder on Synology Drive.
// Parameters:
//   - file_id: The identifier of the folder to list (e.g., MyDrive constant for the root folder)
//
// Returns:
//   - *ListResponseDataV2: Data structure containing the list of items and total count
//   - error: HttpError if there was a network or request error
//   - error: SynologyError if the listing failed or the response was invalid
func (s *SynologySession) List(fileID FileID) (*ListResponse, error) {
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

	httpResponse, err := s.httpGet(endpoint, params)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, HttpError(err.Error())
	}

	var jsonResponse jsonListResponseV2
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return nil, SynologyError(err.Error())
	}
	if !jsonResponse.Success {
		return nil, SynologyError(fmt.Sprintf("List folder failed: [code=%d]", jsonResponse.Err.Code))
	}
	for i := range jsonResponse.Data.Items {
		item := jsonResponse.Data.Items[i]
		if !item.ContentType.isValid() {
			return nil, SynologyError(fmt.Sprintf("Invalid content type: %s", item.ContentType))
		}
		if !item.Type.isValid() {
			return nil, SynologyError(fmt.Sprintf("Invalid type: %s", item.Type))
		}
		for j := range item.SharedWith {
			sharedWith := item.SharedWith[j]
			if !sharedWith.Role.isValid() {
				return nil, SynologyError(fmt.Sprintf("Invalid role: %s", sharedWith.Role))
			}
			if !sharedWith.Type.isValid() {
				return nil, SynologyError(fmt.Sprintf("Invalid type: %s", sharedWith.Type))
			}
		}
	}

	resp := ListResponse{
		Items: make([]ListResponseItem, len(jsonResponse.Data.Items)),
		Total: jsonResponse.Data.Total,
		raw:   body,
	}

	// Convert jsonListResponseItemV2 to ListResponseItem using the conversion method
	for i, item := range jsonResponse.Data.Items {
		resp.Items[i] = item.toListResponseItem()
	}

	return &resp, nil
}

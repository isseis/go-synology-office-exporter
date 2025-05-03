package synology_drive_api

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// jsonSharedWithItem represents a user or group that a file or folder is shared with
type jsonSharedWithItem struct {
	DisplayName  string `json:"display_name"`
	Inherited    bool   `json:"inherited"`
	Name         string `json:"name"`
	Nickname     string `json:"nickname"`
	PermissionID string `json:"permission_id"`
	Role         Role   `json:"role"`
	Type         string `json:"type"` // e.g., "user"
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
		UID         int    `json:"uid"`
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
	Type         string
}

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
		UID         int
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
		}
	}

	resp := ListResponse{
		Items: make([]ListResponseItem, len(jsonResponse.Data.Items)),
		Total: jsonResponse.Data.Total,
		raw:   body,
	}
	// Convert jsonListResponseItemV2 to ListResponseItem
	// and populate the ListResponse struct
	for i, item := range jsonResponse.Data.Items {
		resp.Items[i] = ListResponseItem{
			AccessTime: time.Unix(item.AccessTime, 0),
			AdvShared:  item.AdvShared,
			AppProperties: struct {
				Type string
			}{
				Type: item.AppProperties.Type,
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
				CanComment:  item.Capabilities.CanComment,
				CanDelete:   item.Capabilities.CanDelete,
				CanDownload: item.Capabilities.CanDownload,
				CanEncrypt:  item.Capabilities.CanEncrypt,
				CanOrganize: item.Capabilities.CanOrganize,
				CanPreview:  item.Capabilities.CanPreview,
				CanRead:     item.Capabilities.CanRead,
				CanRename:   item.Capabilities.CanRename,
				CanShare:    item.Capabilities.CanShare,
				CanSync:     item.Capabilities.CanSync,
				CanWrite:    item.Capabilities.CanWrite,
			},
			ChangeID:               item.ChangeID,
			ChangeTime:             time.Unix(item.ChangeTime, 0),
			ContentSnippet:         item.ContentSnippet,
			ContentType:            item.ContentType,
			CreatedTime:            time.Unix(item.CreatedTime, 0),
			DisableDownload:        item.DisableDownload,
			DisplayPath:            item.DisplayPath,
			DsmPath:                item.DsmPath,
			EnableWatermark:        item.EnableWatermark,
			Encrypted:              item.Encrypted,
			FileID:                 item.FileID,
			ForceWatermarkDownload: item.ForceWatermarkDownload,
			Hash:                   item.Hash,
			ImageMetadata: struct {
				Time time.Time
			}{
				Time: time.Unix(item.ImageMetadata.Time, 0),
			},
			Labels:       item.Labels,
			MaxID:        item.MaxID,
			ModifiedTime: time.Unix(item.ModifiedTime, 0),
			Name:         item.Name,
			Owner: struct {
				DisplayName string
				Name        string
				Nickname    string
				UID         int
			}{
				DisplayName: item.Owner.DisplayName,
				Name:        item.Owner.Name,
				Nickname:    item.Owner.Nickname,
				UID:         item.Owner.UID,
			},
			ParentID:      item.ParentID,
			Path:          item.Path,
			PermanentLink: item.PermanentLink,
			Properties: struct {
				ObjectID string
			}{
				ObjectID: item.Properties.ObjectID,
			},
			Removed:          item.Removed,
			Revisions:        item.Revisions,
			Shared:           item.Shared,
			SharedWith:       convertSharedWithItems(item.SharedWith),
			Size:             item.Size,
			Starred:          item.Starred,
			SupportRemote:    item.SupportRemote,
			SyncID:           item.SyncID,
			SyncToDevice:     item.SyncToDevice,
			Transient:        item.Transient,
			Type:             item.Type,
			VersionID:        item.VersionID,
			WatermarkVersion: item.WatermarkVersion,
		}
	}

	return &resp, nil
}

// convertSharedWithItems converts a slice of jsonSharedWithItem to a slice of SharedWithItem
func convertSharedWithItems(items []jsonSharedWithItem) []SharedWithItem {
	result := make([]SharedWithItem, len(items))
	for i, item := range items {
		result[i] = SharedWithItem(item)
	}
	return result
}

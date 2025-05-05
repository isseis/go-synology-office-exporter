package synology_drive_api

import (
	"fmt"
	"time"
)

// jsonListResponseItemV2 represents a file or folder item in a Synology Drive listing
// in the raw JSON API response
type jsonListResponseItemV2 struct {
	AccessTime    int64 `json:"access_time"`
	AdvShared     bool  `json:"adv_shared"`
	AppProperties struct {
		Type string `json:"type"`
	} `json:"app_properties"`
	Capabilities           jsonCapabilities `json:"capabilities"`
	ChangeID               int64            `json:"change_id"`
	ChangeTime             int64            `json:"change_time"`
	ContentSnippet         string           `json:"content_snippet"`
	ContentType            contentType      `json:"content_type"`
	CreatedTime            int64            `json:"created_time"`
	DisableDownload        bool             `json:"disable_download"`
	DisplayPath            string           `json:"display_path"`
	DsmPath                string           `json:"dsm_path"`
	EnableWatermark        bool             `json:"enable_watermark"`
	Encrypted              bool             `json:"encrypted"`
	FileID                 FileID           `json:"file_id"`
	ForceWatermarkDownload bool             `json:"force_watermark_download"`
	Hash                   string           `json:"hash"`
	ImageMetadata          struct {
		Time int64 `json:"time"`
	} `json:"image_metadata"`
	Labels        []string  `json:"labels"`
	MaxID         int64     `json:"max_id"`
	ModifiedTime  int64     `json:"modified_time"`
	Name          string    `json:"name"`
	Owner         jsonOwner `json:"owner"`
	ParentID      FileID    `json:"parent_id"`
	Path          string    `json:"path"`
	PermanentLink string    `json:"permanent_link"`
	Properties    struct {
		ObjectID string `json:"object_id"`
	} `json:"properties"`
	Removed          bool             `json:"removed"`
	Revisions        int64            `json:"revisions"`
	Shared           bool             `json:"shared"`
	SharedWith       []jsonSharedWith `json:"shared_with"`
	Size             int64            `json:"size"`
	Starred          bool             `json:"starred"`
	SupportRemote    bool             `json:"support_remote"`
	SyncID           int64            `json:"sync_id"`
	SyncToDevice     bool             `json:"sync_to_device"`
	Transient        bool             `json:"transient"`
	Type             ObjectType       `json:"type"`
	VersionID        string           `json:"version_id"`
	WatermarkVersion int64            `json:"watermark_version"`
}

func (item *jsonListResponseItemV2) validate() error {
	if !item.ContentType.isValid() {
		return SynologyError(fmt.Sprintf("Invalid content type: %s", item.ContentType))
	}
	if !item.Type.isValid() {
		return SynologyError(fmt.Sprintf("Invalid type: %s", item.Type))
	}
	for j := range item.SharedWith {
		sharedWith := item.SharedWith[j]
		if !sharedWith.Role.isValid() {
			return SynologyError(fmt.Sprintf("Invalid role: %s", sharedWith.Role))
		}
		if !sharedWith.Type.isValid() {
			return SynologyError(fmt.Sprintf("Invalid type: %s", sharedWith.Type))
		}
	}
	return nil
}

// jsonListResponseDataV2 represents the data section of a list response
// containing items and total count
type jsonListResponseDataV2 struct {
	Items []jsonListResponseItemV2 `json:"items"`
	Total int64                    `json:"total"`
}

func (d *jsonListResponseDataV2) validate() error {
	for i := range d.Items {
		if err := d.Items[i].validate(); err != nil {
			return err
		}
	}
	if d.Total < 0 {
		return SynologyError(fmt.Sprintf("Invalid total count: %d", d.Total))
	}
	return nil
}

// jsonListResponseV2 represents the complete response from listing files or folders
type jsonListResponseV2 struct {
	synologyAPIResponse
	Data jsonListResponseDataV2 `json:"data"`
}

// ListResponseItem represents a file or folder item in a Synology Drive listing
// with proper Go types for improved usability
type ListResponseItem struct {
	AccessTime    time.Time
	AdvShared     bool
	AppProperties struct {
		Type string
	}
	Capabilities           Capabilities
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
	Labels        []string
	MaxID         int64
	ModifiedTime  time.Time
	Name          string
	Owner         Owner
	ParentID      FileID
	Path          string
	PermanentLink string
	Properties    struct {
		ObjectID string
	}
	Removed          bool
	Revisions        int64
	Shared           bool
	SharedWith       []SharedWith
	Size             int64
	Starred          bool
	SupportRemote    bool
	SyncID           int64
	SyncToDevice     bool
	Transient        bool
	Type             ObjectType
	VersionID        string
	WatermarkVersion int64
}

// toListResponseItem converts the JSON representation to the Go friendly representation
// with proper types such as time.Time instead of Unix timestamps
func (j *jsonListResponseItemV2) toListResponseItem() *ListResponseItem {
	return &ListResponseItem{
		// Convert Unix timestamp (seconds since epoch) to time.Time
		AccessTime: time.Unix(j.AccessTime, 0),
		AdvShared:  j.AdvShared,
		AppProperties: struct {
			Type string
		}{
			Type: j.AppProperties.Type,
		},
		Capabilities:           Capabilities(j.Capabilities),
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
		Labels:        j.Labels,
		MaxID:         j.MaxID,
		ModifiedTime:  time.Unix(j.ModifiedTime, 0),
		Name:          j.Name,
		Owner:         Owner(j.Owner),
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
		SharedWith:       convertSharedWith(j.SharedWith),
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

// ListResponse represents the complete response from listing files or folders
// with proper Go types for improved usability
type ListResponse struct {
	Items []*ListResponseItem
	Total int64
	raw   []byte // Stores the original raw JSON response
}

// List retrieves the contents of a folder on Synology Drive.
// Parameters:
//   - fileID: The identifier of the folder to list (e.g., MyDrive constant for the root folder)
//
// Returns:
//   - *ListResponse: Data structure containing the list of items and total count
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

	var jsonResponse jsonListResponseV2
	body, err := s.processAPIResponse(httpResponse, &jsonResponse, "List folder")
	if err != nil {
		return nil, err
	}

	if err := jsonResponse.Data.validate(); err != nil {
		return nil, err
	}

	resp := ListResponse{
		Items: make([]*ListResponseItem, len(jsonResponse.Data.Items)),
		Total: jsonResponse.Data.Total,
		raw:   body,
	}

	// Convert jsonListResponseItemV2 to ListResponseItem using the conversion method
	for i, item := range jsonResponse.Data.Items {
		resp.Items[i] = item.toListResponseItem()
	}

	return &resp, nil
}

package synology_drive_api

import (
	"fmt"
	"time"
)

type jsonSharedWithMeResponseItemV2 struct {
	AccessTime             jsonTimeStamp     `json:"access_time"`
	AdvShared              bool              `json:"adv_shared"`
	AppProperties          jsonAppProperties `json:"app_properties"`
	Capabilities           jsonCapabilities  `json:"capabilities"`
	ChangeID               int64             `json:"change_id"`
	ChangeTime             jsonTimeStamp     `json:"change_time"`
	ContentSnippet         string            `json:"content_snippet"`
	ContentType            contentType       `json:"content_type"`
	CreatedTime            jsonTimeStamp     `json:"created_time"`
	DisableDownload        bool              `json:"disable_download"`
	DisplayPath            string            `json:"display_path"`
	DsmPath                string            `json:"dsm_path"`
	EnableWatermark        bool              `json:"enable_watermark"`
	Encrypted              bool              `json:"encrypted"`
	FileID                 FileID            `json:"file_id"`
	ForceWatermarkDownload bool              `json:"force_watermark_download"`
	Hash                   string            `json:"hash"`
	ImageMetadata          jsonImageMetadata `json:"image_metadata"`
	Labels                 []string          `json:"labels"`
	MaxID                  int64             `json:"max_id"`
	ModifiedTime           jsonTimeStamp     `json:"modified_time"`
	Name                   string            `json:"name"`
	Owner                  jsonOwner         `json:"owner"`
	ParentID               FileID            `json:"parent_id"`
	Path                   string            `json:"path"`
	PermanentLink          string            `json:"permanent_link"`
	Properties             jsonProperties    `json:"properties"`
	Removed                bool              `json:"removed"`
	Revisions              int               `json:"revisions"`
	Shared                 bool              `json:"shared"`
	SharedWith             []jsonSharedWith  `json:"shared_with"`
	Size                   int64             `json:"size"`
	Starred                bool              `json:"starred"`
	SupportRemote          bool              `json:"support_remote"`
	SyncID                 int64             `json:"sync_id"`
	SyncToDevice           bool              `json:"sync_to_device"`
	Transient              bool              `json:"transient"`
	Type                   ObjectType        `json:"type"`
	VersionID              string            `json:"version_id"`
	WatermarkVersion       int               `json:"watermark_version"`
}

type jsonSharedWithMeResponseDataV2 struct {
	Items []jsonSharedWithMeResponseItemV2 `json:"items"`
	Total int64                            `json:"total"`
}

type jsonSharedWithMeResponseV2 struct {
	synologyAPIResponse
	Data jsonSharedWithMeResponseDataV2 `json:"data"`
}

type SharedWithMeResponseItem struct {
	AccessTime             time.Time
	AdvShared              bool
	AppProperties          AppProperties
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
	ImageMetadata          ImageMetadata
	Labels                 []string
	MaxID                  int64
	ModifiedTime           time.Time
	Name                   string
	Owner                  Owner
	ParentID               FileID
	Path                   string
	PermanentLink          string
	Properties             Properties
	Removed                bool
	Revisions              int
	Shared                 bool
	SharedWith             []SharedWith
	Size                   int64
	Starred                bool
	SupportRemote          bool
	SyncID                 int64
	SyncToDevice           bool
	Transient              bool
	Type                   ObjectType
	VersionID              string
	WatermarkVersion       int
}

func (j *jsonSharedWithMeResponseItemV2) toSharedWithMeResponseItem() *SharedWithMeResponseItem {
	return &SharedWithMeResponseItem{
		AccessTime:             j.AccessTime.toTime(),
		AdvShared:              j.AdvShared,
		AppProperties:          j.AppProperties.toAppProperties(),
		Capabilities:           j.Capabilities.toCapabilities(),
		ChangeID:               j.ChangeID,
		ChangeTime:             j.ChangeTime.toTime(),
		ContentSnippet:         j.ContentSnippet,
		ContentType:            j.ContentType,
		CreatedTime:            j.CreatedTime.toTime(),
		DisableDownload:        j.DisableDownload,
		DisplayPath:            j.DisplayPath,
		DsmPath:                j.DsmPath,
		EnableWatermark:        j.EnableWatermark,
		Encrypted:              j.Encrypted,
		FileID:                 j.FileID,
		ForceWatermarkDownload: j.ForceWatermarkDownload,
		Hash:                   j.Hash,
		ImageMetadata:          j.ImageMetadata.toImageMetadata(),
		Labels:                 j.Labels,
		MaxID:                  j.MaxID,
		ModifiedTime:           j.ModifiedTime.toTime(),
		Name:                   j.Name,
		Owner:                  j.Owner.toOwner(),
		ParentID:               j.ParentID,
		Path:                   j.Path,
		PermanentLink:          j.PermanentLink,
		Properties:             j.Properties.toProperties(),
		Removed:                j.Removed,
		Revisions:              j.Revisions,
		Shared:                 j.Shared,
		SharedWith:             convertSharedWithList(j.SharedWith),
		Size:                   j.Size,
		Starred:                j.Starred,
		SupportRemote:          j.SupportRemote,
		SyncID:                 j.SyncID,
		SyncToDevice:           j.SyncToDevice,
		Transient:              j.Transient,
		Type:                   j.Type,
		VersionID:              j.VersionID,
		WatermarkVersion:       j.WatermarkVersion,
	}
}

type SharedWithMeResponse struct {
	Items []*SharedWithMeResponseItem
	Total int64
	raw   []byte
}

func (s *SynologySession) SharedWithMe() (*SharedWithMeResponse, error) {
	req := apiRequest{
		api:     APINameSynologyDriveFiles,
		method:  "shared_with_me",
		version: "2",
		params: map[string]string{
			"filter":         "{}",
			"sort_direction": "asc",
			"sort_by":        "owner",
			"offset":         "0",
			"limit":          "1000",
		},
	}

	var jsonResponse jsonSharedWithMeResponseV2
	body, err := s.callAPI(req, &jsonResponse, "shared-with-me")
	if err != nil {
		return nil, fmt.Errorf("failed to get shared-with-me contents: %w", err)
	}

	resp := SharedWithMeResponse{
		Total: jsonResponse.Data.Total,
		Items: make([]*SharedWithMeResponseItem, len(jsonResponse.Data.Items)),
		raw:   body,
	}

	for i, item := range jsonResponse.Data.Items {
		resp.Items[i] = item.toSharedWithMeResponseItem()
	}

	return &resp, nil
}

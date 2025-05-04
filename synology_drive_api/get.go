package synology_drive_api

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// jsonGetResponseDataV3 represents the data specific to a file or folder item in a Synology Drive get response
type jsonGetResponseDataV3 struct {
	AccessTime    int64 `json:"access_time"`
	AdvShared     bool  `json:"adv_shared"`
	AppProperties struct {
		Type string `json:"type"`
	} `json:"app_properties"`
	Capabilities           jsonCapabilities `json:"capabilities"`
	ChangeID               int              `json:"change_id"`
	ChangeTime             int64            `json:"change_time"`
	ContentSnippet         string           `json:"content_snippet"`
	ContentType            string           `json:"content_type"`
	CreatedTime            int64            `json:"created_time"`
	DisableDownload        bool             `json:"disable_download"`
	DisplayPath            string           `json:"display_path"`
	DsmPath                string           `json:"dsm_path"`
	EnableWatermark        bool             `json:"enable_watermark"`
	Encrypted              bool             `json:"encrypted"`
	FileID                 string           `json:"file_id"`
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
	Type             FileType         `json:"type"`
	VersionID        string           `json:"version_id"`
	WatermarkVersion int64            `json:"watermark_version"`
}

type jsonGetResponseV3 struct {
	synologyAPIResponse
	Data jsonGetResponseDataV3 `json:"data"`
}

type GetResponse struct {
	AccessTime    time.Time
	AdvShared     bool
	AppProperties struct {
		Type string
	}
	Capabilities           Capabilities
	ChangeID               int
	ChangeTime             time.Time
	ContentSnippet         string
	ContentType            string
	CreatedTime            time.Time
	DisableDownload        bool
	DisplayPath            string
	DsmPath                string
	EnableWatermark        bool
	Encrypted              bool
	FileID                 string
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
	Type             FileType
	VersionID        string
	WatermarkVersion int64

	raw []byte
}

func (j *jsonGetResponseDataV3) toResponse() *GetResponse {
	return &GetResponse{
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

func (s *SynologySession) Get(fileID FileID) (*GetResponse, error) {
	endpoint := "entry.cgi"
	params := map[string]string{
		"api":     "SYNO.SynologyDrive.Files",
		"method":  "get",
		"version": "3",
		"path":    "id:" + string(fileID),
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

	var jsonResponse jsonGetResponseV3
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return nil, SynologyError(err.Error())
	}
	if !jsonResponse.Success {
		return nil, SynologyError(fmt.Sprintf("Get failed: [code=%d]", jsonResponse.Err.Code))
	}

	resp := jsonResponse.Data.toResponse()
	resp.raw = body
	return resp, nil
}

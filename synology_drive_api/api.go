// package synology_drive_api provides functionality to interact with the Synology Drive API
package synology_drive_api

import (
	"strings"
	"time"
)

// SynologyResponse is an interface that provides common response functionality from the Synology API
type SynologyResponse interface {
	GetSuccess() bool
	GetError() synologyError
}

// synologyAPIResponse represents the common structure of all Synology API responses
type synologyAPIResponse struct {
	Success bool          `json:"success"`
	Err     synologyError `json:"error"`
}

// GetSuccess returns the success status of the response
func (r synologyAPIResponse) GetSuccess() bool {
	return r.Success
}

// GetError returns the error information from the response
func (r synologyAPIResponse) GetError() synologyError {
	return r.Err
}

// synologyError represents the error structure in Synology API responses
type synologyError struct {
	Code   int `json:"code"`
	Errors struct {
		Line    int    `json:"line"`
		Message string `json:"message"`
	}
}

// SessionID represents the session identifier for a Synology session,
// which is issued by the Synology API upon successful login
// and is used for subsequent API calls.
type SessionID string

// DeviceID represents a unique identifier for a Synology devices
type DeviceID string

// UserID represents an identifier of a user on SynologyDrive
type UserID int

// FileID represents an identifier of a file on SynologyDrive
// This can be a path (e.g. "/mydrive/somefile.odoc") or an ID (e.g. "123456789")
type FileID string

// isdigits checks if a string contains only digits
func isdigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || '9' < c {
			return false
		}
	}
	return true
}

// toAPIParam converts a FileID to a string that can be used as a parameter in an API request
func (fileID FileID) toAPIParam() string {
	if isdigits(string(fileID)) {
		return "id:" + string(fileID)
	}
	return string(fileID)
}

// FileHash represents the hash of a file on SynologyDrive
type FileHash string

// APIName represents the name of a Synology API
// Used for type safety when specifying the api field in apiRequest
// Example: APINameSynologyDriveFiles, APINameSynologyAPIAuth, etc.
type APIName string

// API names for Synology API requests
const (
	APINameSynologyDriveFiles       APIName = "SYNO.SynologyDrive.Files"
	APINameSynologyDriveTeamFolders APIName = "SYNO.SynologyDrive.TeamFolders"
	APINameSynologyAPIAuth          APIName = "SYNO.API.Auth"
	APINameSynologyOfficeExport     APIName = "SYNO.Office.Export"
)

func isValidAPIName(api APIName) bool {
	switch api {
	case APINameSynologyDriveFiles, APINameSynologyDriveTeamFolders, APINameSynologyAPIAuth, APINameSynologyOfficeExport:
		return true
	default:
		return false
	}
}

func StringToAPIName(s string) APIName {
	if !isValidAPIName(APIName(s)) {
		return ""
	}
	return APIName(s)
}

// MyDrive represents the root folder identifier in Synology Drive
const MyDrive = FileID("/mydrive/")

// ObjectType represents the type of file or directory in Synology Drive
type ObjectType string

// ObjectTypeFile represents a file object in Synology Drive
const ObjectTypeFile = ObjectType("file")

// ObjectTypeDirectory represents a directory object in Synology Drive
const ObjectTypeDirectory = ObjectType("dir")

// contentType represents the type of content in a Synology Drive file
type contentType string

// ContentTypeDocument represents a audio type in Synology Drive
const ContentTypeAudio = contentType("audio")

// ContentTypeDocument represents a document type in Synology Drive
const ContentTypeDocument = contentType("document")

// ContentTypeDirectory represents a directory type in Synology Drive
const ContentTypeDirectory = contentType("dir")

// ContentTypeFile represents a regular file type in Synology Drive
const ContentTypeFile = contentType("file")

// ContentTypeImage represents an image type in Synology Drive
const ContentTypeImage = contentType("image")

// ContentTypeVideo represents a video type in Synology Drive
const ContentTypeVideo = contentType("video")

// isValid checks if the contentType is a valid supported type
func (c contentType) isValid() bool {
	switch c {
	case ContentTypeAudio, ContentTypeDocument, ContentTypeDirectory, ContentTypeFile, ContentTypeImage, ContentTypeVideo:
		return true
	default:
		return false
	}
}

// Role represents the permission level a user has on a shared file or folder
type Role string

// RolePreviewer represents a user who can only preview files
const RolePreviewer = Role("previewer")

// RolePreviewCommenter represents a user who can preview and comment on files
const RolePreviewCommenter = Role("preview_commenter")

// RoleViewer represents a user who can view files
const RoleViewer = Role("viewer")

// RoleCommenter represents a user who can view and comment on files
const RoleCommenter = Role("commenter")

// RoleEditor represents a user who can edit files
const RoleEditor = Role("editor")

// RoleManager represents a user who can manage files and permissions
const RoleManager = Role("organizer")

// SharedEntity represents the type of entity a file can be shared with
type SharedEntity string

// SharedTargetUser represents sharing with a specific user
const SharedTargetUser = SharedEntity("user")

// SharedTargetGroup represents sharing with a group of users
const SharedTargetGroup = SharedEntity("group")

// jsonSharedWith represents a user or group that a file or folder is shared with
// in the raw JSON API response
type jsonSharedWith struct {
	DisplayName  string       `json:"display_name"`
	Inherited    bool         `json:"inherited"`
	Name         string       `json:"name"`
	Nickname     string       `json:"nickname"`
	PermissionID string       `json:"permission_id"`
	Role         Role         `json:"role"`
	Type         SharedEntity `json:"type"` // "user" or "group"
}

// SharedWith represents a user or group that a file or folder is shared with
// in a Go-friendly format
type SharedWith struct {
	DisplayName  string
	Inherited    bool
	Name         string
	Nickname     string
	PermissionID string
	Role         Role
	Type         SharedEntity // "user" or "group"
}

// convertSharedWithList converts a slice of jsonSharedWithItem to a slice of SharedWithItem
func convertSharedWithList(items []jsonSharedWith) []SharedWith {
	result := make([]SharedWith, len(items))
	for i, item := range items {
		result[i] = item.toSharedWith()
	}
	return result
}

// toSharedWith converts a jsonSharedWithItem to a SharedWithItem
func (j *jsonSharedWith) toSharedWith() SharedWith {
	return SharedWith(*j)
}

// jsonAppProperties represents the app properties of a file or folder
// in the raw JSON API response
type jsonAppProperties struct {
	Type string `json:"type"`
}

// AppProperties represents the app properties of a file or folder
type AppProperties struct {
	Type string
}

// toAppProperties converts a jsonAppProperties to an AppProperties
func (j *jsonAppProperties) toAppProperties() AppProperties {
	return AppProperties(*j)
}

// jsonCapabilities represents the permission capabilities a user has on a file or folder
// in the raw JSON API response
type jsonCapabilities struct {
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
}

// Capabilities represents the permission capabilities a user has on a file or folder
// in a Go-friendly format
type Capabilities struct {
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

// toCapabilities converts a jsonCapabilities to a Capabilities
func (j *jsonCapabilities) toCapabilities() Capabilities {
	return Capabilities(*j)
}

// jsonImageMetadata represents the image metadata of a file or folder
// in the raw JSON API response
type jsonImageMetadata struct {
	Time jsonTimeStamp `json:"time"`
}

// ImageMetadata represents the image metadata of a file or folder
type ImageMetadata struct {
	Time time.Time
}

// toImageMetadata converts a jsonImageMetadata to an ImageMetadata
func (j *jsonImageMetadata) toImageMetadata() ImageMetadata {
	return ImageMetadata{
		Time: j.Time.toTime(),
	}
}

// jsonOwner represents the owner information of a file or folder
// in the raw JSON API response
type jsonOwner struct {
	DisplayName string `json:"display_name"`
	Name        string `json:"name"`
	Nickname    string `json:"nickname"`
	UID         UserID `json:"uid"`
}

// Owner represents the owner information of a file or folder
// in a Go-friendly format
type Owner struct {
	DisplayName string
	Name        string
	Nickname    string
	UID         UserID
}

// toOwner converts a jsonOwner to an Owner
func (j *jsonOwner) toOwner() Owner {
	return Owner(*j)
}

// jsonProperties represents the properties of a file or folder
// in the raw JSON API response
type jsonProperties struct {
	ObjectID string `json:"object_id"`
}

// Properties represents the properties of a file or folder
type Properties struct {
	ObjectID string
}

// toProperties converts a jsonProperties to a Properties
func (j *jsonProperties) toProperties() Properties {
	return Properties{
		ObjectID: j.ObjectID,
	}
}

// jsonTimeStamp represents a Unix timestamp in the raw JSON API response
type jsonTimeStamp int64

// toTime converts a jsonTimeStamp to a time.Time
func (j jsonTimeStamp) toTime() time.Time {
	return time.Unix(int64(j), 0)
}

// jsonResponseItem represents a file or folder item in a Synology Drive listing or shared-with-me API response
// in the raw JSON API response
// This type unifies jsonListResponseItemV2 and jsonSharedWithMeResponseItemV2
type jsonResponseItem struct {
	Type                   ObjectType        `json:"type"`
	FileID                 FileID            `json:"file_id"`
	DisplayPath            string            `json:"display_path"`
	Name                   string            `json:"name"`
	ParentID               FileID            `json:"parent_id"`
	Path                   string            `json:"path"`
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
	DsmPath                string            `json:"dsm_path"`
	EnableWatermark        bool              `json:"enable_watermark"`
	Encrypted              bool              `json:"encrypted"`
	ForceWatermarkDownload bool              `json:"force_watermark_download"`
	Hash                   FileHash          `json:"hash"`
	ImageMetadata          jsonImageMetadata `json:"image_metadata"`
	Labels                 []string          `json:"labels"`
	MaxID                  int64             `json:"max_id"`
	ModifiedTime           jsonTimeStamp     `json:"modified_time"`
	Owner                  jsonOwner         `json:"owner"`
	PermanentLink          string            `json:"permanent_link"`
	Properties             jsonProperties    `json:"properties"`
	Removed                bool              `json:"removed"`
	Revisions              int64             `json:"revisions"`
	Shared                 bool              `json:"shared"`
	SharedWith             []jsonSharedWith  `json:"shared_with"`
	Size                   int64             `json:"size"`
	Starred                bool              `json:"starred"`
	SupportRemote          bool              `json:"support_remote"`
	SyncID                 int64             `json:"sync_id"`
	SyncToDevice           bool              `json:"sync_to_device"`
	Transient              bool              `json:"transient"`
	VersionID              string            `json:"version_id"`
	WatermarkVersion       int64             `json:"watermark_version"`
}

// ResponseItem represents a file or folder item in a Synology Drive listing or shared-with-me API response
// with proper Go types for improved usability. This type unifies ListResponseItem and SharedWithMeResponseItem.
// ResponseItem represents a file or folder item in a Synology Drive listing or shared-with-me API response
// with proper Go types for improved usability.
// This type unifies ListResponseItem and SharedWithMeResponseItem
type ResponseItem struct {
	Type        ObjectType
	FileID      FileID
	DisplayPath string
	Name        string
	ParentID    FileID
	Path        string

	AccessTime   time.Time
	ChangeTime   time.Time
	CreatedTime  time.Time
	ModifiedTime time.Time

	AdvShared              bool
	AppProperties          AppProperties
	Capabilities           Capabilities
	ChangeID               int64
	ContentSnippet         string
	ContentType            contentType
	DisableDownload        bool
	DsmPath                string
	EnableWatermark        bool
	Encrypted              bool
	ForceWatermarkDownload bool
	Hash                   FileHash
	ImageMetadata          ImageMetadata
	Labels                 []string
	MaxID                  int64
	Owner                  Owner
	PermanentLink          string
	Properties             Properties
	Removed                bool
	Revisions              int64
	Shared                 bool
	SharedWith             []SharedWith
	Size                   int64
	Starred                bool
	SupportRemote          bool
	SyncID                 int64
	SyncToDevice           bool
	Transient              bool
	VersionID              string
	WatermarkVersion       int64
}

// toResponseItem converts the JSON representation to the Go friendly representation
// with proper types such as time.Time instead of Unix timestamps
func (j *jsonResponseItem) toResponseItem() *ResponseItem {
	return &ResponseItem{
		Type:        j.Type,
		FileID:      j.FileID,
		DisplayPath: j.DisplayPath,
		Name:        j.Name,
		ParentID:    j.ParentID,
		Path:        j.Path,

		AccessTime:   j.AccessTime.toTime(),
		ChangeTime:   j.ChangeTime.toTime(),
		CreatedTime:  j.CreatedTime.toTime(),
		ModifiedTime: j.ModifiedTime.toTime(),

		AdvShared:              j.AdvShared,
		AppProperties:          j.AppProperties.toAppProperties(),
		Capabilities:           j.Capabilities.toCapabilities(),
		ChangeID:               j.ChangeID,
		ContentSnippet:         j.ContentSnippet,
		ContentType:            j.ContentType,
		DisableDownload:        j.DisableDownload,
		DsmPath:                j.DsmPath,
		EnableWatermark:        j.EnableWatermark,
		Encrypted:              j.Encrypted,
		ForceWatermarkDownload: j.ForceWatermarkDownload,
		Hash:                   j.Hash,
		ImageMetadata:          j.ImageMetadata.toImageMetadata(),
		Labels:                 j.Labels,
		MaxID:                  j.MaxID,
		Owner:                  j.Owner.toOwner(),
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
		VersionID:              j.VersionID,
		WatermarkVersion:       j.WatermarkVersion,
	}
}

// officeExtensionMap defines the mapping between Synology Office file extensions
// and their Microsoft Office equivalents
var officeExtensionMap = map[string]string{
	".odoc":    ".docx", // Synology Document to Word
	".osheet":  ".xlsx", // Synology Spreadsheet to Excel
	".oslides": ".pptx", // Synology Presentation to PowerPoint
}

// GetExportFileName converts a Synology Office file name to the equivalent
// Microsoft Office file name based on its extension.
// Returns an empty string if the file format is not a supported Synology Office format.
func GetExportFileName(fileName string) string {
	for synoExt, msExt := range officeExtensionMap {
		if strings.HasSuffix(fileName, synoExt) {
			return strings.TrimSuffix(fileName, synoExt) + msExt
		}
	}
	return ""
}

// package synology_drive_api provides functionality to interact with the Synology Drive API
package synology_drive_api

import "strings"

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
type FileID string

// FileHash represents the hash of a file on SynologyDrive
type FileHash string

// MyDrive represents the root folder identifier in Synology Drive
const MyDrive = FileID("/mydrive/")

// ObjectType represents the type of file or directory in Synology Drive
type ObjectType string

// ObjectTypeFile represents a file object in Synology Drive
const ObjectTypeFile = ObjectType("file")

// ObjectTypeDirectory represents a directory object in Synology Drive
const ObjectTypeDirectory = ObjectType("dir")

// isValid checks if the FileType is a valid supported type
func (o ObjectType) isValid() bool {
	switch o {
	case ObjectTypeFile, ObjectTypeDirectory:
		return true
	default:
		return false
	}
}

// contentType represents the type of content in a Synology Drive file
type contentType string

// ContentTypeDocument represents a document type in Synology Drive
const ContentTypeDocument = contentType("document")

// ContentTypeDirectory represents a directory type in Synology Drive
const ContentTypeDirectory = contentType("dir")

// ContentTypeFile represents a regular file type in Synology Drive
const ContentTypeFile = contentType("file")

// isValid checks if the contentType is a valid supported type
func (c contentType) isValid() bool {
	switch c {
	case ContentTypeDocument, ContentTypeDirectory, ContentTypeFile:
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

// isValid checks if the Role is a valid supported role
func (r Role) isValid() bool {
	switch r {
	case RoleCommenter, RolePreviewer, RolePreviewCommenter, RoleViewer, RoleEditor, RoleManager:
		return true
	default:
		return false
	}
}

// SharedEntity represents the type of entity a file can be shared with
type SharedEntity string

// SharedTargetUser represents sharing with a specific user
const SharedTargetUser = SharedEntity("user")

// SharedTargetGroup represents sharing with a group of users
const SharedTargetGroup = SharedEntity("group")

// isValid checks if the SharedEntity is a valid supported entity type
func (s SharedEntity) isValid() bool {
	switch s {
	case SharedTargetUser, SharedTargetGroup:
		return true
	default:
		return false
	}
}

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

// convertSharedWith converts a slice of jsonSharedWithItem to a slice of SharedWithItem
func convertSharedWith(items []jsonSharedWith) []SharedWith {
	result := make([]SharedWith, len(items))
	for i, item := range items {
		result[i] = SharedWith(item)
	}
	return result
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

// officeExtensionMap defines the mapping between Synology Office file extensions
// and their Microsoft Office equivalents
var officeExtensionMap = map[string]string{
	".odoc":    ".docx", // Synology Document to Word
	".osheet":  ".xlsx", // Synology Spreadsheet to Excel
	".oslides": ".pptx", // Synology Presentation to PowerPoint
}

// getExportFileName converts a Synology Office file name to the equivalent
// Microsoft Office file name based on its extension.
// Returns an empty string if the file format is not a supported Synology Office format.
func getExportFileName(fileName string) string {
	for synoExt, msExt := range officeExtensionMap {
		if strings.HasSuffix(fileName, synoExt) {
			return strings.TrimSuffix(fileName, synoExt) + msExt
		}
	}
	return ""
}

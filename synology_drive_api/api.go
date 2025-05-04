package synology_drive_api

// synologyAPIResponse represents the common structure of all Synology API responses
type synologyAPIResponse struct {
	Success bool `json:"success"`
	Err     struct {
		Code int `json:"code"`
	} `json:"error"`
}

// UserID represents an identifier of a user on SynologyDrive
type UserID int

// FileID represents an identifier of a file on SynologyDrive
type FileID string

const MyDrive = FileID("/mydrive/")

// FileType represents the type of file or directory in Synology Drive
type FileType string

const TypeFile = FileType("file")
const TypeDirectory = FileType("dir")

func (t FileType) isValid() bool {
	switch t {
	case TypeFile, TypeDirectory:
		return true
	default:
		return false
	}
}

// contentType represents the type of content in a Synology Drive file
type contentType string

const ContentTypeDocument = contentType("document")
const ContentTypeDirectory = contentType("dir")
const ContentTypeFile = contentType("file")

func (c contentType) isValid() bool {
	switch c {
	case ContentTypeDocument, ContentTypeDirectory, ContentTypeFile:
		return true
	default:
		return false
	}
}

type Role string

const RolePreviewer = Role("previewer")
const RolePreviewCommenter = Role("preview_commenter")
const RoleViewer = Role("viewer")
const RoleCommenter = Role("commenter")
const RoleEditor = Role("editor")
const RoleManager = Role("organizer")

func (r Role) isValid() bool {
	switch r {
	case RoleCommenter, RolePreviewer, RolePreviewCommenter, RoleViewer, RoleEditor, RoleManager:
		return true
	default:
		return false
	}
}

type SharedEntity string

const SharedTargetUser = SharedEntity("user")
const SharedTargetGroup = SharedEntity("group")

func (s SharedEntity) isValid() bool {
	switch s {
	case SharedTargetUser, SharedTargetGroup:
		return true
	default:
		return false
	}
}

// jsonSharedWith represents a user or group that a file or folder is shared with
type jsonSharedWith struct {
	DisplayName  string       `json:"display_name"`
	Inherited    bool         `json:"inherited"`
	Name         string       `json:"name"`
	Nickname     string       `json:"nickname"`
	PermissionID string       `json:"permission_id"`
	Role         Role         `json:"role"`
	Type         SharedEntity `json:"type"` // "user" or "group"
}

// SharedWithItem represents a user or group that a file or folder is shared with
type SharedWith struct {
	DisplayName  string
	Inherited    bool
	Name         string
	Nickname     string
	PermissionID string
	Role         Role
	Type         SharedEntity // "user" or "group"
}

// convertSharedWithItems converts a slice of jsonSharedWithItem to a slice of SharedWithItem
func convertSharedWith(items []jsonSharedWith) []SharedWith {
	result := make([]SharedWith, len(items))
	for i, item := range items {
		result[i] = SharedWith(item)
	}
	return result
}

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

type jsonOwner struct {
	DisplayName string `json:"display_name"`
	Name        string `json:"name"`
	Nickname    string `json:"nickname"`
	UID         UserID `json:"uid"`
}

type Owner struct {
	DisplayName string
	Name        string
	Nickname    string
	UID         UserID
}

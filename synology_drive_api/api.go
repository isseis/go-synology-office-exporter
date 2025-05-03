package synology_drive_api

// synologyAPIResponse represents the common structure of all Synology API responses
type synologyAPIResponse struct {
	Success bool `json:"success"`
	Err     struct {
		Code int `json:"code"`
	} `json:"error"`
}

// FileID represents an identifier of a file on SynologyDrive
type FileID string

const MyDrive = FileID("/mydrive/")

// Type represents the type of file or directory in Synology Drive
type Type string

const TypeFile = Type("file")
const TypeDirectory = Type("dir")

func (t Type) isValid() bool {
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

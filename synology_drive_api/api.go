package synology_drive_api

// synologyAPIResponse represents the common structure of all Synology API responses
type synologyAPIResponse struct {
	Success bool `json:"success"`
	Err     struct {
		Code int `json:"code"`
	} `json:"error"`
}

// SynologyDriveFileID represents an identifier of a file on SynologyDrive
type SynologyDriveFileID string

const MyDrive = SynologyDriveFileID("/mydrive/")

// contentType represents the type of content in a Synology Drive file
type contentType string

const ContentTypeDocument = contentType("document")

func (c contentType) isValid() bool {
	switch c {
	case ContentTypeDocument:
		return true
	default:
		return false
	}
}

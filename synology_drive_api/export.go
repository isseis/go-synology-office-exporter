package synology_drive_api

import "errors"

// ExportResponse represents the response from exporting a file from Synology Drive
type ExportResponse struct{}

// Export exports a file from Synology Drive with the given file ID
// This function is currently not implemented
// Parameters:
//   - fileID: The identifier of the file to export
//
// Returns:
//   - *ExportResponse: A response containing export details (currently nil)
//   - error: An error indicating that the function is not implemented
func (s *SynologySession) Export(fileID FileID) (*ExportResponse, error) {
	return nil, errors.New("not implemented")
}

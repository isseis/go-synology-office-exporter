package synology_drive_api

import (
	"fmt"
	"io"
)

// ExportResponse represents the response from exporting a file from Synology Drive
type ExportResponse struct {
	Name    string // The name of the exported file
	Content []byte
}

// Export exports a Synology Office file from Synology Drive and converts it to the equivalent Microsoft Office format
// It first retrieves the file information using the Get method, then exports the file using the SYNO.Office.Export API
// Parameters:
//   - fileID: The identifier of the file to export
//
// Returns:
//   - *ExportResponse: A response containing the exported file content as raw bytes
//   - error: An error if the export operation failed, including unsupported file types
func (s *SynologySession) Export(fileID FileID) (*ExportResponse, error) {
	ret, err := s.Get(fileID)
	if err != nil {
		return nil, SynologyError(err.Error())
	}

	exportName := getExportFileName(ret.Name)
	if exportName == "" {
		return nil, SynologyError(fmt.Sprintf("Unsupported file type: [name=%s]", ret.Name))
	}

	endpoint := fmt.Sprintf("entry.cgi/%s", exportName)
	params := map[string]string{
		"api":     "SYNO.Office.Export",
		"method":  "download",
		"version": "1",
		"path":    fmt.Sprintf("id:%s", ret.FileID),
	}

	// Use httpGetWithOptions with empty ContentType for export operations
	httpResponse, err := s.httpGet(endpoint, params, RequestOption{})
	if err != nil {
		return nil, err
	}

	defer httpResponse.Body.Close()
	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, HttpError(err.Error())
	}

	resp := &ExportResponse{
		Name:    exportName,
		Content: body,
	}
	return resp, nil
}

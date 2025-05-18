package synology_drive_api

import (
	"fmt"
	"io"
)

// ExportResponse contains the result of exporting a file from Synology Drive, including the file name and raw content.
type ExportResponse struct {
	Name    string // The name of the exported file
	Content []byte
}

// Export retrieves and converts a Synology Office file to the Microsoft Office format.
//   - fileID: The identifier of the file to export.
//   - Returns an ExportResponse with the exported file content, or an error if the operation fails or the file type is unsupported.
func (s *SynologySession) Export(fileID FileID) (*ExportResponse, error) {
	ret, err := s.Get(fileID)
	if err != nil {
		return nil, SynologyError(err.Error())
	}

	exportName := GetExportFileName(ret.Name)
	if exportName == "" {
		return nil, SynologyError(fmt.Sprintf("Unsupported file type: [name=%s]", ret.Name))
	}

	endpoint := fmt.Sprintf("entry.cgi/%s", exportName)
	params := map[string]string{
		"api":     string(APINameSynologyOfficeExport),
		"method":  "download",
		"version": "1",
		"path":    ret.FileID.toAPIParam(),
	}

	// Use httpGet with empty ContentType for export operations
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

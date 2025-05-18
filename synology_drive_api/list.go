package synology_drive_api

import (
	"fmt"
)

type jsonListResponseDataV2 struct {
	Items []jsonResponseItem `json:"items"`
	Total int64              `json:"total"`
}
type jsonListResponseV2 struct {
	synologyAPIResponse
	Data jsonListResponseDataV2 `json:"data"`
}

type ListResponse struct {
	Items []*ResponseItem
	Total int64
	raw   []byte
}

// List retrieves the contents of a folder on Synology Drive.
//   - fileID: The identifier of the folder to list (e.g., MyDrive for the root folder).
//   - Returns a ListResponse with all items and total count, or an error if the operation fails.
func (s *SynologySession) List(fileID FileID) (*ListResponse, error) {
	req := apiRequest{
		api:     APINameSynologyDriveFiles,
		method:  "list",
		version: "2",
		params: map[string]string{
			"filter":         "{}",
			"sort_direction": "asc",
			"sort_by":        "owner",
			"offset":         "0",
			"limit":          "1000",
			"path":           fileID.toAPIParam(),
		},
	}

	var jsonResponse jsonListResponseV2
	body, err := s.callAPI(req, &jsonResponse, "List folder")
	if err != nil {
		return nil, fmt.Errorf("failed to list folder %s: %w", fileID, err)
	}

	resp := ListResponse{
		Items: make([]*ResponseItem, len(jsonResponse.Data.Items)),
		Total: jsonResponse.Data.Total,
		raw:   body,
	}

	for i, item := range jsonResponse.Data.Items {
		resp.Items[i] = item.toResponseItem()
	}

	return &resp, nil
}

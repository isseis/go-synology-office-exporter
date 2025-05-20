package synology_drive_api

import (
	"fmt"
	"strconv"
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

// List retrieves a paginated list of items from a folder on Synology Drive.
//   - fileID: The identifier of the folder to list (e.g., MyDrive for the root folder)
//   - offset: The starting position (must be >= 0)
//   - limit: Maximum number of items to return (must be > 0 and <= session's maxPageSize)
//   - Returns a ListResponse with items and total count, or an error if the operation fails.
func (s *SynologySession) List(fileID FileID, offset, limit int) (*ListResponse, error) {
	if offset < 0 {
		return nil, fmt.Errorf("offset must be >= 0, got %d", offset)
	}
	if limit <= 0 || limit > s.maxPageSize {
		return nil, fmt.Errorf("limit must be between 1 and %d, got %d", s.maxPageSize, limit)
	}

	req := apiRequest{
		api:     APINameSynologyDriveFiles,
		method:  "list",
		version: "2",
		params: map[string]string{
			"filter":         "{}",
			"sort_direction": "asc",
			"sort_by":        "owner",
			"offset":         strconv.Itoa(offset),
			"limit":          strconv.Itoa(limit),
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

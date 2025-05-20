package synology_drive_api

import (
	"fmt"
	"strconv"
)

// jsonResponseItem represents a file or folder item in a Synology Drive listing or shared-with-me API response
// This type unifies jsonListResponseItemV2 and jsonResponseItemV2
// (definition is in list.go)
// type jsonResponseItem is now defined in list.go

type jsonSharedWithMeResponseDataV2 struct {
	Items []jsonResponseItem `json:"items"`
	Total int64              `json:"total"`
}

type jsonSharedWithMeResponseV2 struct {
	synologyAPIResponse
	Data jsonSharedWithMeResponseDataV2 `json:"data"`
}

type SharedWithMeResponse struct {
	Items []*ResponseItem
	Total int64
	raw   []byte
}

// SharedWithMe retrieves a paginated list of files and folders shared with the user.
//   - offset: The starting position (0-based)
//   - limit: Maximum number of items to return (1-DefaultMaxPageSize)
//   - Returns a SharedWithMeResponse containing the list of shared items and their details,
//     or an error if the API request fails.
func (s *SynologySession) SharedWithMe(offset, limit int) (*SharedWithMeResponse, error) {
	// Validate pagination parameters
	if offset < 0 {
		return nil, fmt.Errorf("offset must be >= 0, got %d", offset)
	}
	if limit <= 0 || limit > DefaultMaxPageSize {
		return nil, fmt.Errorf("limit must be between 1 and %d, got %d", DefaultMaxPageSize, limit)
	}

	req := apiRequest{
		api:     APINameSynologyDriveFiles,
		method:  "shared_with_me",
		version: "2",
		params: map[string]string{
			"filter":         "{}",
			"sort_direction": "asc",
			"sort_by":        "owner",
			"offset":         strconv.Itoa(offset),
			"limit":          strconv.Itoa(limit),
		},
	}

	var jsonResponse jsonSharedWithMeResponseV2
	body, err := s.callAPI(req, &jsonResponse, "shared-with-me")
	if err != nil {
		return nil, fmt.Errorf("failed to get shared-with-me contents: %w", err)
	}

	resp := SharedWithMeResponse{
		Total: jsonResponse.Data.Total,
		Items: make([]*ResponseItem, len(jsonResponse.Data.Items)),
		raw:   body,
	}

	for i, item := range jsonResponse.Data.Items {
		resp.Items[i] = item.toResponseItem()
	}

	return &resp, nil
}

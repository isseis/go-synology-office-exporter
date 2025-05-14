package synology_drive_api

import (
	"fmt"
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

// ResponseItem represents a file or folder item in a Synology Drive listing or shared-with-me API response
// with proper Go types for improved usability. This type unifies ListResponseItem and ResponseItem.
// ResponseItem represents a file or folder item in a Synology Drive listing or shared-with-me API response
// with proper Go types for improved usability. This type unifies ListResponseItem and ResponseItem.
// (definition is in list.go)
// type ResponseItem is now defined in list.go

// toResponseItem converts the JSON representation to the Go friendly representation
// with proper types such as time.Time instead of Unix timestamps
// toResponseItem method is defined in list.go

type SharedWithMeResponse struct {
	Items []*ResponseItem
	Total int64
	raw   []byte
}

func (s *SynologySession) SharedWithMe() (*SharedWithMeResponse, error) {
	req := apiRequest{
		api:     APINameSynologyDriveFiles,
		method:  "shared_with_me",
		version: "2",
		params: map[string]string{
			"filter":         "{}",
			"sort_direction": "asc",
			"sort_by":        "owner",
			"offset":         "0",
			"limit":          "1000",
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

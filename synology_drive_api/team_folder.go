package synology_drive_api

// jsonTeamFolderListItemV1 represents the JSON structure of a team folder list item
type jsonTeamFolderListItemV1 struct {
	Capabilities     jsonCapabilities `json:"capabilities"`
	DisableDownload  bool             `json:"disable_download"`
	EnableVersioning bool             `json:"enable_versioning"`
	FileID           FileID           `json:"file_id"`
	KeepVersions     int              `json:"keep_versions"`
	Name             string           `json:"name"`
	TeamID           string           `json:"team_id"`
}

// jsonTeamFolderListResponseV1 represents the JSON structure of a team folder list response
type jsonTeamFolderListResponseV1 struct {
	synologyAPIResponse
	Data struct {
		Items []jsonTeamFolderListItemV1 `json:"items"`
		Total int                        `json:"total"`
	} `json:"data"`
}

// TeamFolderResponseItem represents a team folder item in a Synology Drive listing
type TeamFolderResponseItem struct {
	Capabilities     Capabilities
	DisableDownload  bool
	EnableVersioning bool
	FileID           FileID
	KeepVersions     int
	Name             string
	TeamID           string
}

// TeamFolderResponse represents the response from listing team folders
type TeamFolderResponse struct {
	Items []*TeamFolderResponseItem
	Total int
	raw   []byte // Stores the original raw JSON response
}

// jsonTeamFolderListItemV1.toTeamFolderResponseItem converts a JSON team folder list item
// to a TeamFolderResponseItem
func (j *jsonTeamFolderListItemV1) toTeamFolderResponseItem() *TeamFolderResponseItem {
	return &TeamFolderResponseItem{
		Capabilities:     Capabilities(j.Capabilities),
		DisableDownload:  j.DisableDownload,
		EnableVersioning: j.EnableVersioning,
		FileID:           j.FileID,
		KeepVersions:     j.KeepVersions,
		Name:             j.Name,
		TeamID:           j.TeamID,
	}
}

// TeamFolder retrieves a list of team folders from the Synology Drive API.
// It returns a TeamFolderResponse containing the list of team folders and their details,
// or an error if the API request fails.
func (s *SynologySession) TeamFolder() (*TeamFolderResponse, error) {
	req := apiRequest{
		api:     APINameSynologyDriveTeamFolders,
		method:  "list",
		version: "1",
		params: map[string]string{
			"filter":         "{}",
			"sort_direction": "asc",
			"sort_by":        "owner",
			"offset":         "0",
			"limit":          "1000",
		},
	}

	var jsonResponse jsonTeamFolderListResponseV1
	body, err := s.callAPI(req, &jsonResponse, "List team folder")
	if err != nil {
		return nil, err
	}

	resp := TeamFolderResponse{
		Items: make([]*TeamFolderResponseItem, len(jsonResponse.Data.Items)),
		Total: jsonResponse.Data.Total,
		raw:   body,
	}
	for i, item := range jsonResponse.Data.Items {
		resp.Items[i] = item.toTeamFolderResponseItem()
	}

	return &resp, nil
}

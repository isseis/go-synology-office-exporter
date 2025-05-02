package synology_drive_api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type SynologySession struct {
	username    string // Username for login on Synology NAS
	password    string // Password for login on Synology NAS
	hostname    string // Hostname of Synology NAS
	scheme      string // URL scheme (http or https)
	sid         string // session id (set after login)
	http_client http.Client
}

// Synology Session name constant for API calls (private to this package)
const synologySessionName = "SynologyDrive"

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

// loginResponseDataV3 represents the data specific to a login response
type loginResponseDataV3 struct {
	DID string `json:"did"`
	SID string `json:"sid"`
}

// loginResponseV3 represents the response from the Synology API after login.
type loginResponseV3 struct {
	synologyAPIResponse
	Data loginResponseDataV3 `json:"data"`
}

// logoutResponseV3 represents the response from the Synology API after logout.
type logoutResponseV3 struct {
	synologyAPIResponse
}

// listResponseV2 represents a file or folder item in a Synology Drive listing
type ListResponseItemV2 struct {
	AccessTime    int64 `json:"access_time"`
	AdvShared     bool  `json:"adv_shared"`
	AppProperties struct {
		Type string `json:"type"`
	} `json:"app_properties"`
	Capabilities struct {
		CanComment  bool `json:"can_comment"`
		CanDelete   bool `json:"can_delete"`
		CanDownload bool `json:"can_download"`
		CanEncrypt  bool `json:"can_encrypt"`
		CanOrganize bool `json:"can_organize"`
		CanPreview  bool `json:"can_preview"`
		CanRead     bool `json:"can_read"`
		CanRename   bool `json:"can_rename"`
		CanShare    bool `json:"can_share"`
		CanSync     bool `json:"can_sync"`
		CanWrite    bool `json:"can_write"`
	} `json:"capabilities"`
	ChangeID               int64               `json:"change_id"`
	ChangeTime             int64               `json:"change_time"`
	ContentSnippet         string              `json:"content_snippet"`
	ContentType            contentType         `json:"content_type"`
	CreatedTime            int64               `json:"created_time"`
	DisableDownload        bool                `json:"disable_download"`
	DisplayPath            string              `json:"display_path"`
	DsmPath                string              `json:"dsm_path"`
	EnableWatermark        bool                `json:"enable_watermark"`
	Encrypted              bool                `json:"encrypted"`
	FileID                 SynologyDriveFileID `json:"file_id"`
	ForceWatermarkDownload bool                `json:"force_watermark_download"`
	Hash                   string              `json:"hash"`
	ImageMetadata          struct {
		Time int64 `json:"time"`
	} `json:"image_metadata"`
	Labels       []string `json:"labels"`
	MaxID        int64    `json:"max_id"`
	ModifiedTime int64    `json:"modified_time"`
	Name         string   `json:"name"`
	Owner        struct {
		DisplayName string `json:"display_name"`
		Name        string `json:"name"`
		Nickname    string `json:"nickname"`
		UID         int    `json:"uid"`
	} `json:"owner"`
	ParentID      string `json:"parent_id"`
	Path          string `json:"path"`
	PermanentLink string `json:"permanent_link"`
	Properties    struct {
		ObjectID string `json:"object_id"`
	} `json:"properties"`
	Removed          bool   `json:"removed"`
	Revisions        int    `json:"revisions"`
	Shared           bool   `json:"shared"`
	SharedWith       []any  `json:"shared_with"`
	Size             int64  `json:"size"`
	Starred          bool   `json:"starred"`
	SupportRemote    bool   `json:"support_remote"`
	SyncID           int64  `json:"sync_id"`
	SyncToDevice     bool   `json:"sync_to_device"`
	Transient        bool   `json:"transient"`
	Type             string `json:"type"`
	VersionID        string `json:"version_id"`
	WatermarkVersion int    `json:"watermark_version"`
}

type ListResponseDataV2 struct {
	Items []ListResponseItemV2 `json:"items"`
	Total int64                `json:"total"`
}

type ListResponseV2 struct {
	synologyAPIResponse
	Data ListResponseDataV2 `json:"data"`
}

// NewSynologySession creates a new Synology API session with the provided credentials and base URL.
// It returns a pointer to the session and an error if the base URL is invalid.
// Parameters:
//   - username: Username for login on Synology NAS
//   - password: Password for login on Synology NAS
//   - base_url: Base URL for the Synology NAS (e.g., "https://nas.example.com:5001")
//
// Returns:
//   - *SynologySession: A new session object
//   - error: An error of type InvalidUrlError if the URL is invalid
func NewSynologySession(username string, password string, base_url string) (*SynologySession, error) {
	parsed, err := url.Parse(base_url)
	if err != nil {
		return nil, InvalidUrlError(err.Error())
	}
	jar, _ := cookiejar.New(nil)
	return &SynologySession{
		username:    username,
		password:    password,
		hostname:    parsed.Host,
		scheme:      parsed.Scheme,
		http_client: http.Client{Jar: jar},
	}, nil
}

func (s *SynologySession) sessionExpired() bool {
	return s.sid == ""
}

func (s *SynologySession) buildUrl(endpoint string, params map[string]string) *url.URL {
	url := &url.URL{
		Scheme: s.scheme,
		Host:   s.hostname,
		Path:   "webapi/" + endpoint,
	}
	query := url.Query()
	for param, value := range params {
		query.Set(param, value)
	}
	if !s.sessionExpired() {
		query.Set("_sid", s.sid)
	}
	url.RawQuery = query.Encode()
	return url
}

func (s *SynologySession) httpGet(endpoint string, params map[string]string) (*http.Response, error) {
	url := s.buildUrl(endpoint, params)
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, HttpError(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.http_client.Do(req)
	if err != nil {
		return nil, HttpError(err.Error())
	}
	return res, nil
}

// Login authenticates with the Synology NAS using the session credentials.
// This stores the session ID for subsequent requests.
// Returns:
//   - error: HttpError if there was a network or request error
//   - error: SynologyError if authentication failed or the response was invalid
func (s *SynologySession) Login() error {
	endpoint := "auth.cgi"
	params := map[string]string{
		"api":     "SYNO.API.Auth",
		"method":  "login",
		"version": "3",
		"account": s.username,
		"passwd":  s.password,
		"session": synologySessionName,
		"format":  "cookie",
	}

	rawResp, err := s.httpGet(endpoint, params)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(rawResp.Body)
	if err != nil {
		return HttpError(err.Error())
	}
	defer rawResp.Body.Close()

	var resp loginResponseV3
	if err := json.Unmarshal(body, &resp); err != nil {
		return SynologyError(err.Error())
	}
	if !resp.Success {
		return SynologyError(fmt.Sprintf("Login failed: [code=%d]", resp.Err.Code))
	}
	sid := resp.Data.SID
	if sid == "" {
		return SynologyError("Invalid or missing 'sid' field in response")
	}

	s.sid = sid
	return nil
}

// Logout terminates the current session on the Synology NAS.
// This clears the session ID for subsequent requests.
// Returns:
//   - error: HttpError if there was a network or request error
//   - error: SynologyError if the logout failed or the response was invalid
func (s *SynologySession) Logout() error {
	endpoint := "auth.cgi"
	params := map[string]string{
		"api":     "SYNO.API.Auth",
		"method":  "logout",
		"version": "3",
		"session": synologySessionName,
	}

	rawResp, err := s.httpGet(endpoint, params)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(rawResp.Body)
	if err != nil {
		return HttpError(err.Error())
	}
	defer rawResp.Body.Close()

	var resp logoutResponseV3
	if err := json.Unmarshal(body, &resp); err != nil {
		return SynologyError(err.Error())
	}
	if !resp.Success {
		return SynologyError(fmt.Sprintf("Logout failed: [code=%d]", resp.Err.Code))
	}
	s.sid = ""
	return nil
}

// List retrieves the contents of a folder on Synology Drive.
// Parameters:
//   - file_id: The identifier of the folder to list (e.g., MyDrive constant for the root folder)
//
// Returns:
//   - *ListResponseDataV2: Data structure containing the list of items and total count
//   - error: HttpError if there was a network or request error
//   - error: SynologyError if the listing failed or the response was invalid
func (s *SynologySession) List(fileID SynologyDriveFileID) (*ListResponseDataV2, error) {
	endpoint := "entry.cgi"
	params := map[string]string{
		"api":            "SYNO.SynologyDrive.Files",
		"method":         "list",
		"version":        "2",
		"filter":         "{}",
		"sort_direction": "asc",
		"sort_by":        "owner",
		"offset":         "0",
		"limit":          "1000",
		"path":           string(fileID),
	}

	rawResp, err := s.httpGet(endpoint, params)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(rawResp.Body)
	if err != nil {
		return nil, HttpError(err.Error())
	}
	defer rawResp.Body.Close()

	var resp ListResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, SynologyError(err.Error())
	}
	if !resp.Success {
		return nil, SynologyError(fmt.Sprintf("List folder failed: [code=%d]", resp.Err.Code))
	}
	for i := range resp.Data.Items {
		item := resp.Data.Items[i]
		if !item.ContentType.isValid() {
			return nil, SynologyError(fmt.Sprintf("Invalid content type: %s", item.Type))
		}
	}
	return &resp.Data, nil
}

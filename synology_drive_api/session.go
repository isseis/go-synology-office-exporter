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

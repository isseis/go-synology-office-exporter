package synology_drive_api

import (
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
	reqUrl := &url.URL{
		Scheme: s.scheme,
		Host:   s.hostname,
		Path:   "webapi/" + endpoint,
	}
	query := reqUrl.Query()
	for param, value := range params {
		query.Set(param, value)
	}
	if !s.sessionExpired() {
		query.Set("_sid", s.sid)
	}
	reqUrl.RawQuery = query.Encode()
	return reqUrl
}

func (s *SynologySession) httpRequest(method string, endpoint string, params map[string]string) (*http.Response, error) {
	url := s.buildUrl(endpoint, params)
	req, err := http.NewRequest(method, url.String(), nil)
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

func (s *SynologySession) httpGet(endpoint string, params map[string]string) (*http.Response, error) {
	return s.httpRequest(http.MethodGet, endpoint, params)
}

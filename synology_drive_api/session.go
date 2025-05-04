package synology_drive_api

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

// SynologySession represents a session with a Synology NAS
type SynologySession struct {
	username    string      // Username for login on Synology NAS
	password    string      // Password for login on Synology NAS
	hostname    string      // Hostname of Synology NAS
	scheme      string      // URL scheme (http or https)
	sid         SessionID   // Session ID (set after login)
	http_client http.Client // HTTP client with cookie support
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

// sessionExpired checks if the session ID is empty, indicating an expired or non-existent session
func (s *SynologySession) sessionExpired() bool {
	return s.sid == ""
}

// buildUrl constructs a URL for an API endpoint with the specified parameters
// Parameters:
//   - endpoint: The API endpoint path
//   - params: Query parameters to include in the URL
//
// Returns:
//   - *url.URL: A URL object with the constructed URL
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
	reqUrl.RawQuery = query.Encode()
	return reqUrl
}

// httpRequest sends an HTTP request to the Synology NAS API
// Parameters:
//   - method: The HTTP method to use (GET, POST, etc.)
//   - endpoint: The API endpoint path
//   - params: Query parameters to include in the URL
//
// Returns:
//   - *http.Response: The HTTP response from the API
//   - error: An error of type HttpError if the request failed
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

// httpGet sends a GET request to the Synology NAS API
// Parameters:
//   - endpoint: The API endpoint path
//   - params: Query parameters to include in the URL
//
// Returns:
//   - *http.Response: The HTTP response from the API
//   - error: An error if the request failed
func (s *SynologySession) httpGet(endpoint string, params map[string]string) (*http.Response, error) {
	return s.httpRequest(http.MethodGet, endpoint, params)
}

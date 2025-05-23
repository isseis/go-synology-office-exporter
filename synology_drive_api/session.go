package synology_drive_api

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

// sleeper is an interface for sleeping, which can be mocked in tests
type sleeper interface {
	Sleep(time.Duration)
}

type realSleeper struct{}

func (s *realSleeper) Sleep(d time.Duration) {
	time.Sleep(d)
}

const (
	defaultMaxRetries = 3
	defaultRetryDelay = 1 * time.Second
)

// RequestOption represents options for HTTP requests
type RequestOption struct {
	ContentType string // Content-Type header value, empty string means no Content-Type header will be set
}

var RequestOptionJSON = RequestOption{
	ContentType: "application/json",
}

// SessionOption configures a SynologySession.
type SessionOption func(*SynologySession)

// WithMaxPageSize sets the maximum number of items that can be requested in a single List call.
// If not set, DefaultMaxPageSize will be used.
func WithMaxPageSize(maxPageSize int64) SessionOption {
	return func(s *SynologySession) {
		s.maxPageSize = maxPageSize
	}
}

// GetMaxPageSize returns the maximum number of items that can be requested per page.
func (s *SynologySession) GetMaxPageSize() int64 {
	return s.maxPageSize
}

// SynologySession represents a session with a Synology NAS
type SynologySession struct {
	username    string      // Username for login on Synology NAS
	password    string      // Password for login on Synology NAS
	hostname    string      // Hostname of Synology NAS
	scheme      string      // URL scheme (http or https)
	sid         SessionID   // Session ID (set after login)
	http_client http.Client // HTTP client with cookie support
	maxPageSize int64       // Maximum number of items per page for List operations
}

// NewSynologySession creates a new Synology API session with the provided credentials and base URL.
// It returns a pointer to the session and an error if the base URL is invalid.
//
// Parameters:
//   - username: Username for login on Synology NAS
//   - password: Password for login on Synology NAS
//   - base_url: Base URL for the Synology NAS (e.g., "https://nas.example.com:5001")
//   - options: Optional configuration functions (e.g., WithMaxPageSize)
//
// Returns:
//   - *SynologySession: A new session object
//   - error: An error of type InvalidUrlError if the URL is invalid
func NewSynologySession(username string, password string, base_url string, options ...SessionOption) (*SynologySession, error) {
	parsed, err := url.Parse(base_url)
	if err != nil {
		return nil, InvalidUrlError(err.Error())
	}
	jar, _ := cookiejar.New(nil)
	session := &SynologySession{
		username:    username,
		password:    password,
		hostname:    parsed.Host,
		scheme:      parsed.Scheme,
		http_client: http.Client{Jar: jar},
		maxPageSize: DefaultMaxPageSize, // Set default max page size
	}

	// Apply all options
	for _, option := range options {
		option(session)
	}

	return session, nil
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
//   - options: Request options including content type settings
//
// Returns:
//   - *http.Response: The HTTP response from the API
//   - error: An error of type HttpError if the request failed
func (s *SynologySession) httpRequest(method string, endpoint string, params map[string]string, options RequestOption) (*http.Response, error) {
	url := s.buildUrl(endpoint, params)
	req, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return nil, HttpError(err.Error())
	}

	// Set Content-Type header only if specified in options
	if options.ContentType != "" {
		req.Header.Set("Content-Type", options.ContentType)
	}

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
//   - options: Request options including content type settings
//
// Returns:
//   - *http.Response: The HTTP response from the API
//   - error: An error if the request failed
func (s *SynologySession) httpGet(endpoint string, params map[string]string, options RequestOption) (*http.Response, error) {
	return s.httpRequest(http.MethodGet, endpoint, params, options)
}

// httpGetJSONDirect sends a GET request to the Synology NAS API with JSON content type without retry
// Parameters:
//   - endpoint: The API endpoint path
//   - params: Query parameters to include in the URL
//
// Returns:
//   - *http.Response: The HTTP response from the API
//   - error: An error if the request failed
func (s *SynologySession) httpGetJSONDirect(endpoint string, params map[string]string) (*http.Response, error) {
	options := RequestOption{
		ContentType: "application/json",
	}
	return s.httpRequest(http.MethodGet, endpoint, params, options)
}

// httpGetJSON sends a GET request to the Synology NAS API with JSON content type and retry logic
// Parameters:
//   - endpoint: The API endpoint path
//   - params: Query parameters to include in the URL
//
// Returns:
//   - *http.Response: The HTTP response from the API
//   - error: An error if all retry attempts failed
func (s *SynologySession) httpGetJSON(endpoint string, params map[string]string) (*http.Response, error) {
	return s.httpGetJSONWithRetry(endpoint, params, defaultMaxRetries, defaultRetryDelay, &realSleeper{})
}

// isRetryableStatus returns true if the HTTP status code is considered retryable.
// Retries on all 5xx errors and selected 4xx errors (Request Timeout, Too Many Requests, Unauthorized, Forbidden).
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout, // 408
		http.StatusTooManyRequests, // 429
		http.StatusUnauthorized,    // 401
		http.StatusForbidden:       // 403
		return true
	}
	return code >= 500 && code < 600
}

// httpGetJSONWithRetry sends a GET request with retry logic
// Retries on network errors, HTTP 5xx (server) errors, and selected 4xx errors (see isRetryableStatus).
// Only sleeps between retries, not before the first attempt.
// Returns the first successful response, or error after all retries.
func (s *SynologySession) httpGetJSONWithRetry(endpoint string, params map[string]string, maxRetries int, retryDelay time.Duration, sleeper sleeper) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Only sleep between retries, not before the first attempt
		if attempt > 0 {
			sleeper.Sleep(retryDelay)
		}

		resp, err := s.httpGetJSONDirect(endpoint, params)
		if err == nil {
			// Retry on HTTP 5xx and selected 4xx errors
			if isRetryableStatus(resp.StatusCode) {
				lastErr = fmt.Errorf("retryable error: %d %s", resp.StatusCode, resp.Status)
				resp.Body.Close()
				continue
			}
			return resp, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("after %d attempts, last error: %w", maxRetries+1, lastErr)
}

// apiRequest represents a Synology API request with its required parameters
type apiRequest struct {
	api     APIName           // API name (e.g., APINameSynologyDriveFiles)
	method  string            // API method (e.g., "list", "get")
	version string            // API version (e.g., "1", "2", "3")
	params  map[string]string // Additional parameters
}

// callAPI handles an API call with required parameters explicitly defined.
// This ensures that the required parameters (api, method, version) are always provided.
// Parameters:
//   - req: apiRequest containing the required parameters and any additional parameters
//   - synRes: Pointer to a struct implementing the SynologyResponse interface to unmarshal the JSON into
//   - errorContext: Context information for error messages (e.g. operation name)
//
// Returns:
//   - []byte: Raw JSON response data
//   - error: Any error encountered during processing
func (s *SynologySession) callAPI(req apiRequest, synRes SynologyResponse, errorContext string) ([]byte, error) {
	// Create a new map with the required parameters
	params := make(map[string]string)

	// Set the required parameters
	params["api"] = string(req.api)
	params["method"] = req.method
	params["version"] = req.version

	// Add any additional parameters
	maps.Copy(params, req.params)

	// Determine the appropriate endpoint based on the API being accessed
	endpoint := "entry.cgi"
	if req.api == APINameSynologyAPIAuth {
		endpoint = "auth.cgi"
	}

	httpResponse, err := s.httpGetJSON(endpoint, params)
	if err != nil {
		return nil, err
	}

	return s.processAPIResponse(httpResponse, synRes, errorContext)
}

// processAPIResponse processes the API response, unmarshals the JSON, and checks if it was successful
// Parameters:
//   - response: HTTP response from the API
//   - synRes: Pointer to a struct implementing the SynologyResponse interface to unmarshal the JSON into
//   - errorContext: Context information for error messages
//
// Returns:
//   - []byte: Raw JSON response data
//   - error: Any error encountered during processing
func (s *SynologySession) processAPIResponse(response *http.Response, synRes SynologyResponse, errorContext string) ([]byte, error) {

	// Read the response body
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, HttpError(err.Error())
	}

	// Unmarshal the JSON
	if err := json.Unmarshal(body, synRes); err != nil {
		return body, SynologyError(err.Error())
	}

	if !synRes.GetSuccess() {
		// Get error information
		err := synRes.GetError()

		if err.Errors.Message != "" {
			return body, SynologyError(fmt.Sprintf("%s failed: %s [code=%d, line=%d]",
				errorContext, err.Errors.Message, err.Code, err.Errors.Line))
		}
		return body, SynologyError(fmt.Sprintf("%s failed: [code=%d]", errorContext, err.Code))
	}

	return body, nil
}

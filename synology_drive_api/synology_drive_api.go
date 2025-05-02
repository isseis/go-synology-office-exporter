package synology_drive_api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
)

type SynologySession struct {
	username      string // Username for login on Synology NAS
	password      string // Password for login on Synology NAS
	hostname      string // Hostname of Synology NAS
	scheme        string // URL scheme (http or https)
	sid           string // session id (set after login)
	sessionExpire bool
	http_client   http.Client
}

// Common Synology API error codes (not exhaustive)
//
// These error codes are used in the Synology API documentation:
// https://global.download.synology.com/download/Document/Software/DeveloperGuide/Os/DSM/All/enu/DSM_Login_Web_API_Guide_enu.pdf
const SYNOLOGY_COMMON_ERROR_UNKNOWN = 100
const SYNOLOGY_COMMON_ERROR_INVALID_PARAMETER = 101
const SYNOLOGY_COMMON_ERROR_API_NOT_EXIST = 102
const SYNOLOGY_COMMON_ERROR_METHOD_NOT_EXIST = 103
const SYNOLOGY_COMMON_ERROR_VERSION_NOT_SUPPORTED = 104
const SYNOLOGY_COMMON_ERROR_DOES_NOT_HAVE_PERMISSION = 105
const SYNOLOGY_COMMON_ERROR_SESSION_TIMEOUT = 106
const SYNOLOGY_COMMON_ERROR_SESSION_INTERRUPTED = 107
const SYNOLOGY_COMMON_ERROR_UPLOAD_FILE_FAILED = 108
const SYNOLOGY_COMMON_ERROR_NETWORK_UNSTABLE_OR_SYSTEM_BUSY = 109
const SYNOLOGY_COMMON_ERROR_NETWORK_UNSTABLE_OR_SYSTEM_BUSY_2 = 110
const SYNOLOGY_COMMON_ERROR_NETWORK_UNSTABLE_OR_SYSTEM_BUSY_3 = 111
const SYNOLOGY_COMMON_ERROR_RESERVED_112 = 112
const SYNOLOGY_COMMON_ERROR_RESERVED_113 = 113
const SYNOLOGY_COMMON_ERROR_LOST_PARAMETER = 114
const SYNOLOGY_COMMON_ERROR_UPLOAD_FILE_DISALLOWED = 115
const SYNOLOGY_COMMON_ERROR_OPERATION_DISALLOWED_FOR_DEMO_SITE = 116
const SYNOLOGY_COMMON_ERROR_NETWORK_UNSTABLE_OR_SYSTEM_BUSY_4 = 117
const SYNOLOGY_COMMON_ERROR_NETWORK_UNSTABLE_OR_SYSTEM_BUSY_5 = 118
const SYNOLOGY_COMMON_ERROR_INVALID_SESSION = 119
const SYNOLOGY_COMMON_ERROR_RESERVED_120 = 120
const SYNOLOGY_COMMON_ERROR_RESERVED_121 = 121
const SYNOLOGY_COMMON_ERROR_RESERVED_122 = 122
const SYNOLOGY_COMMON_ERROR_RESERVED_123 = 123
const SYNOLOGY_COMMON_ERROR_RESERVED_124 = 124
const SYNOLOGY_COMMON_ERROR_RESERVED_125 = 125
const SYNOLOGY_COMMON_ERROR_RESERVED_126 = 126
const SYNOLOGY_COMMON_ERROR_RESERVED_127 = 127
const SYNOLOGY_COMMON_ERROR_RESERVED_128 = 128
const SYNOLOGY_COMMON_ERROR_RESERVED_129 = 129
const SYNOLOGY_COMMON_ERROR_RESERVED_130 = 130
const SYNOLOGY_COMMON_ERROR_RESERVED_131 = 131
const SYNOLOGY_COMMON_ERROR_RESERVED_132 = 132
const SYNOLOGY_COMMON_ERROR_RESERVED_133 = 133
const SYNOLOGY_COMMON_ERROR_RESERVED_134 = 134
const SYNOLOGY_COMMON_ERROR_RESERVED_135 = 135
const SYNOLOGY_COMMON_ERROR_RESERVED_136 = 136
const SYNOLOGY_COMMON_ERROR_RESERVED_137 = 137
const SYNOLOGY_COMMON_ERROR_RESERVED_138 = 138
const SYNOLOGY_COMMON_ERROR_RESERVED_139 = 139
const SYNOLOGY_COMMON_ERROR_RESERVED_140 = 140
const SYNOLOGY_COMMON_ERROR_RESERVED_141 = 141
const SYNOLOGY_COMMON_ERROR_RESERVED_142 = 142
const SYNOLOGY_COMMON_ERROR_RESERVED_143 = 143
const SYNOLOGY_COMMON_ERROR_RESERVED_144 = 144
const SYNOLOGY_COMMON_ERROR_RESERVED_145 = 145
const SYNOLOGY_COMMON_ERROR_RESERVED_146 = 146
const SYNOLOGY_COMMON_ERROR_RESERVED_147 = 147
const SYNOLOGY_COMMON_ERROR_RESERVED_148 = 148
const SYNOLOGY_COMMON_ERROR_RESERVED_149 = 149
const SYNOLOGY_COMMON_ERROR_SOURCE_IP_MISMATCH = 150

// Synlogy Login API error codes
const SYNOLOGY_LOGIN_ERROR_ACCOUNT_OR_PASSWORD_INCORRECT = 400
const SYNOLOGY_LOGIN_ERROR_ACCOUNT_DISABLED = 401
const SYNOLOGY_LOGIN_ERROR_PERMISSION_DENIED = 402
const SYNOLOGY_LOGIN_ERROR_2FA_REQUIRED = 403
const SYNOLOGY_LOGIN_ERROR_2FA_CODE_INCORRECT = 404
const SYNOLOGY_LOGIN_ERROR_RESERVED_405 = 405
const SYNOLOGY_LOGIN_ERROR_2FA_ENFORCED = 406
const SYNOLOGY_LOGIN_ERROR_IP_BLOCKED = 407
const SYNOLOGY_LOGIN_ERROR_CANNOT_CHANGE_EXPIRED_PASSWORD = 408
const SYNOLOGY_LOGIN_ERROR_PASSWORD_EXPIRED = 409
const SYNOLOGY_LOGIN_ERROR_PASSWORD_MUST_BE_CHANGED = 410

// Synology Session name constant for API calls (private to this package)
const synologySessionName = "SynologyDrive"

// SynologyAuthResponse represents the response from the Synology API after authentication.
type synologyAuthResponseV3 struct {
	Success bool `json:"success"`
	Data    struct {
		DID string `json:"did"`
		SID string `json:"sid"`
	} `json:"data"`
	Err struct {
		Code int `json:"code"`
	} `json:"error"`
}

type InvalidUrlError string

func (e InvalidUrlError) Error() string {
	return "invalid URL " + strconv.Quote(string(e)) + " in base_url"
}

type HttpError string

func (e HttpError) Error() string {
	return "http error " + strconv.Quote(string(e))
}

type SynologyError string

func (e SynologyError) Error() string {
	return "synology error " + strconv.Quote(string(e))
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
		username:      username,
		password:      password,
		hostname:      parsed.Host,
		scheme:        parsed.Scheme,
		sessionExpire: true,
		http_client:   http.Client{Jar: jar},
	}, nil
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
	if !s.sessionExpire {
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
// It performs an API call to auth.cgi endpoint and stores the session ID for subsequent requests.
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

	resp, err := s.httpGet(endpoint, params)
	if err != nil {
		return HttpError(err.Error())
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return HttpError(err.Error())
	}
	defer resp.Body.Close()

	var authResponse synologyAuthResponseV3
	if err := json.Unmarshal(body, &authResponse); err != nil {
		return SynologyError(err.Error())
	}
	if !authResponse.Success {
		return SynologyError(fmt.Sprintf("Login failed: [code=%d]", authResponse.Err.Code))
	}
	sid := authResponse.Data.SID
	if sid == "" {
		return SynologyError("Invalid or missing 'sid' field in response")
	}

	s.sid = sid
	s.sessionExpire = false
	return nil
}

// Logout terminates the current session on the Synology NAS.
// It performs an API call to auth.cgi endpoint to invalidate the current session ID.
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

	resp, err := s.httpGet(endpoint, params)
	if err != nil {
		return HttpError(err.Error())
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return HttpError(err.Error())
	}
	defer resp.Body.Close()

	var authResponse synologyAuthResponseV3
	if err := json.Unmarshal(body, &authResponse); err != nil {
		return SynologyError(err.Error())
	}
	if !authResponse.Success {
		return SynologyError(fmt.Sprintf("Logout failed: [code=%d]", authResponse.Err.Code))
	}
	s.sid = ""
	s.sessionExpire = true
	return nil
}

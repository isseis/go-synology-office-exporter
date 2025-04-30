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

func (s *SynologySession) Login(application string) error {
	endpoint := "auth.cgi"
	params := map[string]string{
		"api":     "SYNO.API.Auth",
		"method":  "login",
		"version": "3",
		"account": s.username,
		"passwd":  s.password,
		"session": application,
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

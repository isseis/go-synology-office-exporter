package synology_drive_api

import (
	"encoding/json"
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

type InvalidUrlError string

func (e InvalidUrlError) Error() string {
	return "invalid URL " + strconv.Quote(string(e)) + " in base_url"
}

// TODO: Add detailed information about the error
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
	query.Set("account", s.username)
	query.Set("passwd", s.password)
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
	loginAPIVersion := "3" // Assuming DSM version 7
	params := map[string]string{
		"api":     "SYNO.API.Auth",
		"version": loginAPIVersion,
		"method":  "login",
		"account": s.username,
		"passwd":  s.password,
		"session": application,
		"format":  "cookie",
	}

	resp, err := s.httpGet(endpoint, params)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic("Could not read response body")
	}
	defer resp.Body.Close()

	/*
		The body is expected to be in JSON format like:
		{"data":{"did":"YYYYYYYYYYYYYYYYYYYYYYYYY","sid":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"},"success":true}
	*/

	var parsedBody map[string]interface{}
	if err := json.Unmarshal([]byte(body), &parsedBody); err != nil {
		return SynologyError(err.Error())
	}
	success, ok := parsedBody["success"].(bool)
	if !ok || !success {
		return SynologyError("Login failed")
	}

	data, ok := parsedBody["data"].(map[string]interface{})
	if !ok {
		return SynologyError("Invalid or missing 'data' field in response")
	}

	sid, ok := data["sid"].(string)
	if !ok || sid == "" {
		return SynologyError("Invalid or missing 'sid' field in response")
	}
	s.sid = sid
	s.sessionExpire = false

	return nil
}

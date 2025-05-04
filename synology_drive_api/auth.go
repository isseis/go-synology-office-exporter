package synology_drive_api

import (
	"encoding/json"
	"fmt"
	"io"
)

// Synology Session name constant for API calls (private to this package)
const synologySessionName = "SynologyDrive"

// loginResponseDataV3 represents the data specific to a login response
type loginResponseDataV3 struct {
	DID DeviceID  `json:"did"` // Device ID
	SID SessionID `json:"sid"` // Session ID
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

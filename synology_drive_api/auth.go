package synology_drive_api

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
	req := apiRequest{
		api:     "SYNO.API.Auth",
		method:  "login",
		version: "3",
		params: map[string]string{
			"account": s.username,
			"passwd":  s.password,
			"session": synologySessionName,
			"format":  "cookie",
		},
	}

	var resp loginResponseV3
	_, err := s.callAPI(req, &resp, "Login")
	if err != nil {
		return err
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
	req := apiRequest{
		api:     "SYNO.API.Auth",
		method:  "logout",
		version: "3",
		params: map[string]string{
			"session": synologySessionName,
		},
	}

	var resp logoutResponseV3
	_, err := s.callAPI(req, &resp, "Logout")
	if err != nil {
		return err
	}

	s.sid = ""
	return nil
}

package synology_drive_api

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestMain sets up a mock Synology NAS server by default.
// If USE_REAL_SYNOLOGY is set, tests will run against a real NAS device.
func TestMain(m *testing.M) {
	if os.Getenv("USE_REAL_SYNOLOGY") == "" {
		// Default: use mock
		mockServer := httptest.NewServer(http.HandlerFunc(mockSynologyHandler))
		defer mockServer.Close()
		os.Setenv("SYNOLOGY_NAS_URL", mockServer.URL)
		os.Setenv("SYNOLOGY_NAS_USER", "mock-user")
		os.Setenv("SYNOLOGY_NAS_PASS", "mock-pass")
	}
	os.Exit(m.Run())
}

// mockSynologyHandler handles requests to the mock Synology NAS API.

// mockLoggedIn tracks login state for the mock Synology NAS server.
var mockLoggedIn bool

// ResetMockLogin resets the mock login state to 'not logged in'.
// Call this at the start of each test to avoid state leakage between tests.
func ResetMockLogin() {
	mockLoggedIn = false
}

//go:embed data/files_list_response.json
var cannedResponseListFiles []byte

//go:embed data/files_get_response.json
var cannedResponseGetFile []byte

func mockSynologyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[MOCK] %s %s\n", r.Method, r.URL.String())
	switch r.URL.Path {
	case "/webapi/auth.cgi":
		method := r.URL.Query().Get("method")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		switch method {
		case "login":
			mockLoggedIn = true
			w.Write([]byte(`{"success": true, "data": {"sid": "mock-sid"}}`))
		case "logout":
			mockLoggedIn = false
			w.Write([]byte(`{"success": true}`))
		default:
			w.Write([]byte(`{"success": false, "error": {"code": 100}}`))
		}
	case "/webapi/entry.cgi":
		api := StringToAPIName(r.URL.Query().Get("api"))
		method := r.URL.Query().Get("method")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if !mockLoggedIn {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"success": false, "error": {"code": 119, "errors": {"line": 0, "message": "not logged in"}}}`))
			return
		}

		if api == APINameSynologyDriveFiles && method == "list" {
			w.Write(cannedResponseListFiles)
			return
		} else if method == "get" {
			w.Header().Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
			w.Write(cannedResponseGetFile)
			return
		} else {
			resp := map[string]any{
				"success": true,
				"data":    map[string]any{},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

	default:
		w.Write([]byte(`{"success": true}`))
	}
}

// getEnvOrPanic returns environment variables for test credentials and URL.
// By default, returns mock values unless USE_REAL_SYNOLOGY is set.
func getEnvOrPanic(key string) string {
	if value, exists := os.LookupEnv(key); !exists {
		panic(key + " is not set")
	} else {
		return value
	}
}

func getNasUrl() string {
	return getEnvOrPanic("SYNOLOGY_NAS_URL")
}

func getNasUser() string {
	return getEnvOrPanic("SYNOLOGY_NAS_USER")
}

func getNasPass() string {
	return getEnvOrPanic("SYNOLOGY_NAS_PASS")
}

func TestContentTypeIsValid(t *testing.T) {
	tests := []struct {
		name string
		ct   contentType
		want bool
	}{
		{"ContentTypeDocument", ContentTypeDocument, true},
		{"ContentTypeDirectory", ContentTypeDirectory, true},
		{"ContentTypeFile", ContentTypeFile, true},
		{"Invalid content type", contentType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ct.isValid(); got != tt.want {
				t.Errorf("contentType.isValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetExportFileName(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		want     string
	}{
		{"Document conversion", "test.odoc", "test.docx"},
		{"Spreadsheet conversion", "test.osheet", "test.xlsx"},
		{"Presentation conversion", "test.oslides", "test.pptx"},
		{"Document with path", "/path/to/document.odoc", "/path/to/document.docx"},
		{"File with multiple dots", "my.important.spreadsheet.osheet", "my.important.spreadsheet.xlsx"},
		{"Unsupported extension", "test.txt", ""},
		{"No extension", "test", ""},
		{"Empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetExportFileName(tt.fileName); got != tt.want {
				t.Errorf("getExportFileName(%q) = %q, want %q", tt.fileName, got, tt.want)
			}
		})
	}
}

func TestFileIDToAPIParam(t *testing.T) {
	tests := []struct {
		name  string
		input FileID
		want  string
	}{
		{"All digits", FileID("12345"), "id:12345"},
		{"Non-digits", FileID("abcde"), "abcde"},
		{"Mixed digits and letters", FileID("123abc"), "123abc"},
		{"Empty string", FileID(""), ""},
		{"Leading zeros", FileID("00123"), "id:00123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.toAPIParam()
			if got != tt.want {
				t.Errorf("FileID(%q).toAPIParam() = %q, want %q", string(tt.input), got, tt.want)
			}
		})
	}
}

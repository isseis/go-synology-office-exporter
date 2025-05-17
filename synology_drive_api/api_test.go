package synology_drive_api

import (
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

const cannedResponseListFiles = `
{
	"data": {
		"items": [
			{
				"access_time": 1746239241,
				"adv_shared": false,
				"app_properties": {
					"type": "none"
				},
				"capabilities": {
					"can_comment": true,
					"can_delete": true,
					"can_download": true,
					"can_encrypt": true,
					"can_organize": true,
					"can_preview": true,
					"can_read": true,
					"can_rename": true,
					"can_share": true,
					"can_sync": true,
					"can_write": true
				},
				"change_id": 17,
				"change_time": 1746239241,
				"content_snippet": "",
				"content_type": "dir",
				"created_time": 1746239202,
				"disable_download": false,
				"display_path": "/mydrive/2025-04",
				"dsm_path": "",
				"enable_watermark": false,
				"encrypted": false,
				"file_id": "882614106016756317",
				"force_watermark_download": false,
				"hash": "",
				"image_metadata": {
					"time": 1746239241
				},
					"labels": [],
				"max_id": 14,
				"modified_time": 1746239241,
				"name": "2025-04",
				"owner": {
					"display_name": "backup",
					"name": "backup",
					"nickname": "",
					"uid": 1029
				},
				"parent_id": "880423525918227781",
				"path": "/2025-04",
				"permanent_link": "13CNgI0QlJMjbqcZf5k3HRoZsbCAqejY",
				"properties": {},
				"removed": false,
				"revisions": 1,
				"shared": false,
				"shared_with": [],
				"size": 0,
				"starred": false,
				"support_remote": false,
				"sync_id": 8,
				"sync_to_device": false,
				"transient": false,
				"type": "dir",
				"version_id": "8",
				"watermark_version": 0
			},
			{
				"access_time": 1746239530,
				"adv_shared": false,
				"app_properties": {
					"type": "none"
				},
				"capabilities": {
					"can_comment": true,
					"can_delete": true,
					"can_download": true,
					"can_encrypt": true,
					"can_organize": true,
					"can_preview": true,
					"can_read": true,
					"can_rename": true,
					"can_share": true,
					"can_sync": true,
					"can_write": true
				},
				"change_id": 17,
				"change_time": 1746239530,
				"content_snippet": "",
				"content_type": "document",
				"created_time": 1746239207,
				"disable_download": false,
				"display_path": "/mydrive/planning.osheet",
				"dsm_path": "",
				"enable_watermark": false,
				"encrypted": false,
				"file_id": "882614125167948399",
				"force_watermark_download": false,
				"hash": "7dd71dd1192ca985884d934470fb9d3c",
				"image_metadata": {
					"time": 1746239207
				},
				"labels": [],
				"max_id": 17,
				"modified_time": 1746239207,
				"name": "planning.osheet",
				"owner": {
					"display_name": "backup",
					"name": "backup",
					"nickname": "",
					"uid": 1029
				},
				"parent_id": "880423525918227781",
				"path": "/planning.osheet",
				"permanent_link": "13CNgcuVDEENCkSfDkcKpr39pcK6hhPD",
				"properties": {
					"object_id": "1029_ELSD3CRAC557FDOVULJJJDKN50.sh"
				},
				"removed": false,
				"revisions": 3,
				"shared": false,
				"shared_with": [],
				"size": 720,
				"starred": false,
				"support_remote": false,
				"sync_id": 17,
				"sync_to_device": false,
				"transient": false,
				"type": "file",
				"version_id": "17",
				"watermark_version": 0
			}
		],
		"total": 2
	},
	"success": true
}`

const cannedResponseGetFile = `
{
	"data": {
		"access_time": 1746357311,
		"adv_shared": false,
		"app_properties": {"type": "none"},
		"capabilities": {
			"can_comment": true,
			"can_delete": true,
			"can_download": true,
			"can_encrypt": true,
			"can_organize": true,
			"can_preview": true,
			"can_read": true,
			"can_rename": true,
			"can_share": true,
			"can_sync": true,
			"can_write": true
		},
		"change_id": 17,
		"change_time": 1746239530,
		"content_snippet": "",
		"content_type": "document",
		"created_time": 1746239207,
		"disable_download": false,
		"display_path": "/mydrive/planning.osheet",
		"dsm_path": "",
		"enable_watermark": false,
		"encrypted": false,
		"file_id": "882614125167948399",
		"force_watermark_download": false,
		"hash": "7dd71dd1192ca985884d934470fb9d3c",
		"image_metadata": {"time": 1746239207},
		"labels": [],
		"max_id": 17,
		"modified_time": 1746239207,
		"name": "planning.osheet",
		"owner": {
			"display_name": "backup",
			"name": "backup",
			"nickname": "",
			"uid": 1029
		},
		"parent_id": "880423525918227781",
		"path": "/planning.osheet",
		"permanent_link": "13CNgcuVDEENCkSfDkcKpr39pcK6hhPD",
		"properties": {"object_id": "1029_ELSD3CRAC557FDOVULJJJDKN50.sh"},
		"removed": false,
		"revisions": 3,
		"shared": false,
		"shared_with": [],
		"size": 720,
		"starred": false,
		"support_remote": false,
		"sync_id": 17,
		"sync_to_device": false,
		"transient": false,
		"type": "file",
		"version_id": "17",
		"watermark_version": 0
	},
	"success": true
}`

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
			w.Write([]byte(cannedResponseListFiles))
			return
		} else if method == "get" {
			w.Header().Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
			w.Write([]byte(cannedResponseGetFile))
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

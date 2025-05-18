//go:build !integration
// +build !integration

package synology_drive_api

import "testing"

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

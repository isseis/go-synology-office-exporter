package synology_drive_api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	tests := []struct {
		name          string
		offset        int64
		limit         int64
		expectedError string
	}{
		{"valid request", 0, 100, ""},
		{"negative offset", -1, 100, "offset must be >= 0"},
		{"zero limit", 0, 0, "limit must be between 1 and 1000"},
		{"limit too large", 0, 1001, "limit must be between 1 and 1000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetMockLogin()
			s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
			require.NoError(t, err)
			err = s.Login()
			require.NoError(t, err)

			resp, err := s.List(MyDrive, tt.offset, tt.limit)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			for _, item := range resp.Items {
				assert.NotEmpty(t, item.FileID)
				assert.NotEmpty(t, item.Name)
				assert.NotEmpty(t, item.Path)
			}
		})
	}
}

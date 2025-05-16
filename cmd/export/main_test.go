package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSources(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []sourceType
		wantErr bool
	}{
		{
			name:  "empty string returns all sources",
			input: "",
			want:  []sourceType{sourceMyDrive, sourceTeamFolder, sourceShared},
		},
		{
			name:  "single source",
			input: "mydrive",
			want:  []sourceType{sourceMyDrive},
		},
		{
			name:  "multiple sources",
			input: "mydrive, teamfolder",
			want:  []sourceType{sourceMyDrive, sourceTeamFolder},
		},
		{
			name:  "duplicate sources",
			input: "mydrive, mydrive, shared",
			want:  []sourceType{sourceMyDrive, sourceShared},
		},
		{
			name:    "invalid source",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "case sensitive",
			input:   "MyDrive",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSources(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestDefaultSources(t *testing.T) {
	sources := defaultSources()
	assert.ElementsMatch(t, []sourceType{sourceMyDrive, sourceTeamFolder, sourceShared}, sources)
}

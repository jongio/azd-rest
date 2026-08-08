//go:build mage

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExtensionVersion(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		want     string
		wantErr  string
	}{
		{
			name:     "version",
			manifest: "id: jongio.azd.rest\nversion: 1.2.3\n",
			want:     "1.2.3",
		},
		{
			name:     "quoted version",
			manifest: "version: '1.2.3'\n",
			want:     "1.2.3",
		},
		{
			name:     "missing version",
			manifest: "id: jongio.azd.rest\n",
			wantErr:  "version is required",
		},
		{
			name:     "invalid YAML",
			manifest: "version: [\n",
			wantErr:  "did not find expected node content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExtensionVersion([]byte(tt.manifest))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

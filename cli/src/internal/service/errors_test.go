package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHostFromURL covers the telemetry label derived from a request URL,
// including the malformed case. An unparseable URL must produce an empty label
// rather than a partial one: a bad label would split one service's telemetry
// across several spellings, which is worse than having no label at all.
func TestHostFromURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "https with path", raw: "https://management.azure.com/subscriptions", want: "management.azure.com"},
		{name: "host and port", raw: "https://localhost:8080/v1", want: "localhost:8080"},
		{name: "empty", raw: "", want: ""},
		{name: "unparseable", raw: "http://[::1", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hostFromURL(tt.raw))
		})
	}
}

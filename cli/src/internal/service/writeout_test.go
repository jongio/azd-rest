package service

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
)

// longToken is a fake bearer token long enough (over 12 chars) to exercise the
// redaction path. It is built from a short literal to keep spell-check happy.
var longToken = strings.Repeat("ab", 20)

func newWriteOutResponse() *client.Response {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-Ms-Request-Id", "abc-123")
	h.Set("Authorization", "Bearer "+longToken)
	return &client.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    h,
		Body:       []byte(`{"ok":true}`),
		Duration:   1500 * time.Millisecond,
	}
}

func TestExpandWriteOut_Variables(t *testing.T) {
	resp := newWriteOutResponse()
	tests := []struct {
		name   string
		format string
		want   string
	}{
		{"http_code", "%{http_code}", "200"},
		{"http_status", "%{http_status}", "200 OK"},
		{"time_total", "%{time_total}", "1.500000"},
		{"time_total_ms", "%{time_total_ms}", "1500"},
		{"size_download", "%{size_download}", "11"},
		{"content_type", "%{content_type}", "application/json"},
		{"method", "%{method}", "GET"},
		{"url", "%{url}", "https://example.com/api"},
		{"header", "%{header.X-Ms-Request-Id}", "abc-123"},
		{"combined", "%{http_code} %{size_download}", "200 11"},
		{"escapes", "%{http_code}\\n", "200\n"},
		{"tab", "a\\tb", "a\tb"},
		{"unknown token left literal", "%{bogus}", "%{bogus}"},
		{"absent header empty", "[%{header.X-Missing}]", "[]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandWriteOut(tt.format, "GET", "https://example.com/api", resp)
			if got != tt.want {
				t.Errorf("ExpandWriteOut(%q) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}

func TestExpandWriteOut_RedactsSensitiveHeader(t *testing.T) {
	resp := newWriteOutResponse()
	got := ExpandWriteOut("%{header.Authorization}", "GET", "https://example.com", resp)
	if got == "Bearer "+longToken {
		t.Fatalf("Authorization header was not redacted: %q", got)
	}
	if !strings.Contains(got, "...") {
		t.Fatalf("expected a redacted value containing \"...\", got %q", got)
	}
}

package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHeaderExpectation(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    headerExpectation
		wantErr string
	}{
		{
			name: "presence only",
			raw:  "ETag",
			want: headerExpectation{name: "ETag", original: "ETag"},
		},
		{
			name: "equals value",
			raw:  "Content-Type=application/json",
			want: headerExpectation{name: "Content-Type", value: "application/json", hasValue: true, original: "Content-Type=application/json", separator: "="},
		},
		{
			name: "colon value",
			raw:  "Content-Type: application/json",
			want: headerExpectation{name: "Content-Type", value: "application/json", hasValue: true, original: "Content-Type: application/json", separator: ":"},
		},
		{
			name:    "empty",
			raw:     " ",
			wantErr: "cannot be empty",
		},
		{
			name:    "missing name",
			raw:     "=application/json",
			wantErr: "header name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHeaderExpectation(tt.raw)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckExpectedHeaders(t *testing.T) {
	resp := &client.Response{
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
			"Etag":         []string{`"v1"`},
		},
	}

	require.NoError(t, checkExpectedHeaders(resp, []string{"content-type"}))
	require.NoError(t, checkExpectedHeaders(resp, []string{"Content-Type=application/json"}))
	require.NoError(t, checkExpectedHeaders(resp, []string{"Content-Type: application/json"}))

	err := checkExpectedHeaders(resp, []string{"X-Request-Id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "X-Request-Id")

	err = checkExpectedHeaders(resp, []string{"Content-Type=text/plain"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "application/json")
}

func TestCheckExpectedHeaders_RedactsSensitiveValues(t *testing.T) {
	resp := &client.Response{
		Headers: http.Header{
			"Set-Cookie": []string{"session=super-secret-cookie-value"},
		},
	}

	err := checkExpectedHeaders(resp, []string{"Set-Cookie=expected-secret-cookie"})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "super-secret-cookie-value")
	assert.NotContains(t, err.Error(), "expected-secret-cookie")
}

func TestExecute_ExpectHeaderMismatchWritesBodyThenReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.ExpectedHeaders = []string{"Content-Type=text/plain"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Content-Type")

	out, readErr := os.ReadFile(cfg.OutputFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(out), `"ok": true`)
}

func TestExecute_ExpectHeaderPasses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "req-123")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.ExpectedHeaders = []string{"x-request-id=req-123"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL)
	require.NoError(t, err)
}

func TestExecute_ExpectHeaderParseError(t *testing.T) {
	cfg := baseTestConfig(t)
	cfg.ExpectedHeaders = []string{"=value"}

	err := newTestService().Execute(context.Background(), cfg, "GET", "http://127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "header name is required")
	assert.False(t, errors.Is(err, http.ErrServerClosed))
}

func TestRedactedHeaderValues(t *testing.T) {
	got := redactedHeaderValues("X-Test", []string{"one", "two"})
	assert.True(t, strings.Contains(got, `"one"`))
	assert.True(t, strings.Contains(got, `"two"`))
}

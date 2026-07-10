package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawExitCoder mirrors the structural exit-code contract used by main so tests
// can assert the exit code of a returned error without importing the cmd package.
type rawExitCoder interface{ ExitCode() int }

func TestRawOutputText(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantText string
		wantOK   bool
	}{
		{"string", `"web"`, "web\n", true},
		{"array of strings", `["a","b","c"]`, "a\nb\nc\n", true},
		{"empty array", `[]`, "", true},
		{"number", `5`, "", false},
		{"boolean", `true`, "", false},
		{"null", `null`, "", false},
		{"object", `{"a":1}`, "", false},
		{"mixed array", `["a",1]`, "", false},
		{"invalid json", `not json`, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, ok := rawOutputText([]byte(tc.body))
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantText, text)
		})
	}
}

// TestExecute_RawOutput_MissingQuery verifies --raw-output without --query fails
// with exit code 2 before any network call is made.
func TestExecute_RawOutput_MissingQuery(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.RawOutput = true // no Query set

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL)
	require.Error(t, err)
	assert.False(t, called, "no request should be made when the flag combination is invalid")

	var coder rawExitCoder
	require.True(t, errors.As(err, &coder), "error should carry an exit code")
	assert.Equal(t, 2, coder.ExitCode())
	assert.Contains(t, err.Error(), "--raw-output requires --query")
}

func TestExecute_RawOutput_StringValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"web"}`))
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.Query = "name"
	cfg.RawOutput = true

	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))

	got, err := os.ReadFile(cfg.OutputFile)
	require.NoError(t, err)
	assert.Equal(t, "web\n", string(got))
}

func TestExecute_RawOutput_ArrayOfStrings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"names":["a","b"]}`))
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.Query = "names"
	cfg.RawOutput = true

	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))

	got, err := os.ReadFile(cfg.OutputFile)
	require.NoError(t, err)
	assert.Equal(t, "a\nb\n", string(got))
}

// TestExecute_RawOutput_NonStringFallsBackToJSON verifies a non-string result
// is still written as JSON rather than being mangled by the raw path.
func TestExecute_RawOutput_NonStringFallsBackToJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":5}`))
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.Query = "count"
	cfg.RawOutput = true

	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))

	got, err := os.ReadFile(cfg.OutputFile)
	require.NoError(t, err)
	assert.Equal(t, "5", strings.TrimSpace(string(got)))
}

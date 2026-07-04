package service

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteDiagnostic_NotSilent verifies advisory messages are written when
// silent mode is off.
func TestWriteDiagnostic_NotSilent(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostic(&buf, false, "Warning: %s\n", "disabled")

	got := buf.String()
	if !strings.Contains(got, "Warning: disabled") {
		t.Fatalf("expected diagnostic to be written, got %q", got)
	}
}

// TestWriteDiagnostic_Silent verifies advisory messages are suppressed when
// silent mode is on.
func TestWriteDiagnostic_Silent(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostic(&buf, true, "Warning: %s\n", "disabled")

	if got := buf.String(); got != "" {
		t.Fatalf("expected no diagnostic output in silent mode, got %q", got)
	}
}

// TestWriteDiagnostic_FormatsArgs verifies the helper formats arguments like
// fmt.Fprintf when not silent.
func TestWriteDiagnostic_FormatsArgs(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostic(&buf, false, "> Pagination enabled (max %d pages)\n", 100)

	if got := buf.String(); got != "> Pagination enabled (max 100 pages)\n" {
		t.Fatalf("unexpected formatted output: %q", got)
	}
}

// newTestService builds a RequestService whose token provider is never used
// (tests target http:// endpoints where authentication is skipped).
func newTestService() *RequestService {
	return NewRequestService(
		func() (client.TokenProvider, error) { return nil, nil },
		DefaultHTTPClientFactory,
	)
}

func baseTestConfig(t *testing.T) config.Config {
	t.Helper()
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = filepath.Join(t.TempDir(), "out.json")
	return cfg
}

func TestExecute_MaxTimeExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := baseTestConfig(t)
	cfg.Retry = 0
	cfg.MaxTime = 20 * time.Millisecond

	err := newTestService().Execute(context.Background(), cfg, "GET", server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--max-time")
}

func TestExecute_WithinMaxTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := baseTestConfig(t)
	cfg.MaxTime = 5 * time.Second

	err := newTestService().Execute(context.Background(), cfg, "GET", server.URL)
	require.NoError(t, err)
}

func TestExecute_MaxTimeDisabledByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := baseTestConfig(t)
	cfg.MaxTime = 0

	err := newTestService().Execute(context.Background(), cfg, "GET", server.URL)
	require.NoError(t, err)
}

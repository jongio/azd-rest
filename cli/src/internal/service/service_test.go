package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

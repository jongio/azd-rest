package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// maxLatencyCoder mirrors the structural exit-code contract used by main so the
// tests can assert the exit code without importing the cmd package.
type maxLatencyCoder interface{ ExitCode() int }

func TestParseMaxLatency(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{name: "empty disables", value: "", want: 0},
		{name: "milliseconds", value: "500ms", want: 500 * time.Millisecond},
		{name: "seconds", value: "2s", want: 2 * time.Second},
		{name: "unparseable", value: "later", wantErr: true},
		{name: "zero rejected", value: "0s", wantErr: true},
		{name: "negative rejected", value: "-1s", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMaxLatency(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				var coder maxLatencyCoder
				require.True(t, errors.As(err, &coder), "config error should carry an exit code")
				assert.Equal(t, 2, coder.ExitCode())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExecute_MaxLatency_SlowResponseExits28(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(40 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.json")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.MaxLatency = "5ms"

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/slow")
	require.Error(t, err)

	var coder maxLatencyCoder
	require.True(t, errors.As(err, &coder), "slow response should carry an exit code")
	assert.Equal(t, 28, coder.ExitCode())

	// The body is written before the error is returned.
	out, readErr := os.ReadFile(tmp)
	require.NoError(t, readErr)
	assert.Contains(t, string(out), "ok")
}

func TestExecute_MaxLatency_FastResponseSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = filepath.Join(t.TempDir(), "out.json")
	cfg.MaxLatency = "1m"

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/fast")
	require.NoError(t, err)
}

func TestExecute_MaxLatency_InvalidExitsBeforeRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = filepath.Join(t.TempDir(), "out.json")
	cfg.MaxLatency = "nope"

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/never")
	require.Error(t, err)

	var coder maxLatencyCoder
	require.True(t, errors.As(err, &coder), "invalid value should carry an exit code")
	assert.Equal(t, 2, coder.ExitCode())
	assert.False(t, called, "no request should be made when the budget is invalid")
}

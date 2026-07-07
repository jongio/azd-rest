package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_Redact_JSONMasksField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"kv","value":"s3cr3t"}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.json")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.Redact = []string{"value"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/secret")
	require.NoError(t, err)

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	body := string(out)

	assert.Contains(t, body, "REDACTED")
	assert.NotContains(t, body, "s3cr3t")
	assert.Contains(t, body, "kv")
}

func TestExecute_Redact_RawLeftUnchanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":"s3cr3t"}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.txt")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFormat = "raw"
	cfg.OutputFile = tmp
	cfg.Redact = []string{"value"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/secret")
	require.NoError(t, err)

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	body := string(out)

	assert.Contains(t, body, "s3cr3t")
	assert.NotContains(t, body, "REDACTED")
}

func TestExecute_Redact_TableMasksField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[{"name":"a","secret":"top"}]}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.txt")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFormat = "table"
	cfg.OutputFile = tmp
	cfg.Redact = []string{"value.*.secret"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/list")
	require.NoError(t, err)

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	body := string(out)

	assert.Contains(t, body, "REDACTED")
	assert.NotContains(t, body, "top")
	assert.Contains(t, strings.ToLower(body), "name")
}

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeRedactFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "redact.txt")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadRedactFile_ParsesAndSkipsCommentsAndBlanks(t *testing.T) {
	path := writeRedactFile(t, "# comment\nvalue\n\nnested.token\n   # indented comment\nvalue.*.secret\n")

	got, err := loadRedactFile(path)

	require.NoError(t, err)
	assert.Equal(t, []string{"value", "nested.token", "value.*.secret"}, got)
}

func TestLoadRedactFile_InvalidPathWithWhitespace(t *testing.T) {
	path := writeRedactFile(t, "value\nnot a path\n")

	_, err := loadRedactFile(path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2")
}

func TestExecute_RedactFile_JSONMasksFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"kv","value":"s3cr3t","nested":{"token":"abc"}}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.json")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.RedactFile = writeRedactFile(t, "value\nnested.token\n")

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/secret")
	require.NoError(t, err)

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	body := string(out)

	assert.Contains(t, body, "REDACTED")
	assert.NotContains(t, body, "s3cr3t")
	assert.NotContains(t, body, "abc")
	assert.Contains(t, body, "kv")
}

func TestExecute_RedactFileMissingFailsBeforeRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.RedactFile = filepath.Join(t.TempDir(), "missing.txt")

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/secret")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open redact file")
	assert.False(t, called)
}

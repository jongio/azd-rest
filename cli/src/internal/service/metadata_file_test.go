package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResponseMetadata_RedactsHeaders(t *testing.T) {
	resp := &client.Response{
		Status:     "201 Created",
		StatusCode: http.StatusCreated,
		Duration:   1500 * time.Millisecond,
		Body:       []byte(`{"ok":true}`),
		Headers: http.Header{
			"Content-Type":  []string{"application/json"},
			"Authorization": []string{"******"},
			"Set-Cookie":    []string{"session=super-secret-cookie-value"},
		},
	}

	metadata := newResponseMetadata("POST", "https://example.com/resource", resp)

	assert.Equal(t, "POST", metadata.Method)
	assert.Equal(t, "https://example.com/resource", metadata.URL)
	assert.Equal(t, "201 Created", metadata.Status)
	assert.Equal(t, http.StatusCreated, metadata.StatusCode)
	assert.Equal(t, int64(1500), metadata.DurationMs)
	assert.Equal(t, len(resp.Body), metadata.SizeDownload)
	assert.Equal(t, "application/json", metadata.ContentType)
	assert.NotContains(t, metadata.Headers["Set-Cookie"][0], "super-secret-cookie-value")
}

func TestWriteResponseMetadata_WritesJSONFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	resp := &client.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Duration:   25 * time.Millisecond,
		Body:       []byte(`{"ok":true}`),
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	require.NoError(t, writeResponseMetadata(path, "GET", "https://example.com", resp))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var metadata responseMetadata
	require.NoError(t, json.Unmarshal(data, &metadata))
	assert.Equal(t, "GET", metadata.Method)
	assert.Equal(t, "200 OK", metadata.Status)
	assert.Equal(t, int64(25), metadata.DurationMs)
}

func TestWriteResponseMetadata_WriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "metadata.json")
	resp := &client.Response{Status: "200 OK", StatusCode: http.StatusOK}

	err := writeResponseMetadata(path, "GET", "https://example.com", resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write metadata file")
}

func TestExecute_MetadataFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-123")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.MetadataFile = filepath.Join(t.TempDir(), "metadata.json")

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/items?api-version=2024-01-01")
	require.NoError(t, err)

	data, readErr := os.ReadFile(cfg.MetadataFile)
	require.NoError(t, readErr)

	var metadata responseMetadata
	require.NoError(t, json.Unmarshal(data, &metadata))
	assert.Equal(t, "GET", metadata.Method)
	assert.Equal(t, srv.URL+"/items?api-version=2024-01-01", metadata.URL)
	assert.Equal(t, http.StatusOK, metadata.StatusCode)
	assert.Equal(t, "req-123", metadata.Headers["X-Request-Id"][0])
}

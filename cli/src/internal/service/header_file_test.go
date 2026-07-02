package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testHeaderURL  = "https://management.azure.com/subscriptions"
	testAcceptJSON = "application/json"
)

func writeHeaderFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "headers.txt")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func newHeaderTestService() *RequestService {
	return NewRequestService(
		func() (client.TokenProvider, error) { return nil, nil },
		DefaultHTTPClientFactory,
	)
}

func TestLoadHeaderFile_ParsesAndSkipsCommentsAndBlanks(t *testing.T) {
	path := writeHeaderFile(t, "# a comment\nAccept: application/json\n\nX-Trace: abc123\n   # indented comment\nX-Empty:\n")

	got, err := loadHeaderFile(path)

	require.NoError(t, err)
	assert.Equal(t, testAcceptJSON, got["Accept"])
	assert.Equal(t, "abc123", got["X-Trace"])
	assert.Equal(t, "", got["X-Empty"])
	assert.Len(t, got, 3)
}

func TestLoadHeaderFile_MalformedLine(t *testing.T) {
	path := writeHeaderFile(t, "Accept: application/json\nnot-a-header\n")

	_, err := loadHeaderFile(path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2")
}

func TestLoadHeaderFile_EmptyHeaderName(t *testing.T) {
	path := writeHeaderFile(t, ": value-only\n")

	_, err := loadHeaderFile(path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty header name")
}

func TestLoadHeaderFile_MissingFile(t *testing.T) {
	_, err := loadHeaderFile(filepath.Join(t.TempDir(), "does-not-exist.txt"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open header file")
}

func TestBuildRequestOptions_HeaderFileLoadsHeaders(t *testing.T) {
	path := writeHeaderFile(t, "Accept: application/json\nX-Trace: abc123\n")
	cfg := config.Config{NoAuth: true, HeaderFile: path}

	opts, cleanup, err := newHeaderTestService().BuildRequestOptions(cfg, "GET", testHeaderURL)
	if cleanup != nil {
		defer cleanup()
	}

	require.NoError(t, err)
	assert.Equal(t, testAcceptJSON, opts.Headers["Accept"])
	assert.Equal(t, "abc123", opts.Headers["X-Trace"])
}

func TestBuildRequestOptions_InlineHeaderOverridesFile(t *testing.T) {
	path := writeHeaderFile(t, "Accept: application/xml\n")
	cfg := config.Config{
		NoAuth:     true,
		HeaderFile: path,
		Headers:    []string{"Accept: application/json"},
	}

	opts, cleanup, err := newHeaderTestService().BuildRequestOptions(cfg, "GET", testHeaderURL)
	if cleanup != nil {
		defer cleanup()
	}

	require.NoError(t, err)
	assert.Equal(t, testAcceptJSON, opts.Headers["Accept"])
}

func TestBuildRequestOptions_HeaderFileMissingReturnsError(t *testing.T) {
	cfg := config.Config{NoAuth: true, HeaderFile: filepath.Join(t.TempDir(), "nope.txt")}

	_, cleanup, err := newHeaderTestService().BuildRequestOptions(cfg, "GET", testHeaderURL)
	if cleanup != nil {
		defer cleanup()
	}

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open header file")
}

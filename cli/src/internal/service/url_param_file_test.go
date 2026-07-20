package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeURLParamFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "params.txt")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadURLParamFile_ParsesAndSkipsCommentsAndBlanks(t *testing.T) {
	path := writeURLParamFile(t, "# a comment\napi-version=2024-01-01\n\n$top=10\n   # indented comment\nempty=\n")

	got, err := loadURLParamFile(path)

	require.NoError(t, err)
	assert.Equal(t, []string{"api-version=2024-01-01", "$top=10", "empty="}, got)
}

func TestLoadURLParamFile_MalformedLine(t *testing.T) {
	path := writeURLParamFile(t, "api-version=2024-01-01\nnot-a-param\n")

	_, err := loadURLParamFile(path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2")
}

func TestLoadURLParamFile_EmptyParamName(t *testing.T) {
	path := writeURLParamFile(t, "=value-only\n")

	_, err := loadURLParamFile(path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty parameter name")
}

func TestBuildRequestOptions_URLParamFileLoadsParams(t *testing.T) {
	path := writeURLParamFile(t, "api-version=2024-01-01\n$top=10\n")
	cfg := config.Config{NoAuth: true, URLParamFile: path}

	opts, cleanup, err := newHeaderTestService().BuildRequestOptions(cfg, "GET", testHeaderURL)
	if cleanup != nil {
		defer cleanup()
	}

	require.NoError(t, err)
	assert.Equal(t, "https://management.azure.com/subscriptions?%24top=10&api-version=2024-01-01", opts.URL)
}

func TestBuildRequestOptions_InlineURLParamOverridesFile(t *testing.T) {
	path := writeURLParamFile(t, "filter=all\ntag=file\n")
	cfg := config.Config{
		NoAuth:       true,
		URLParamFile: path,
		URLParams:    []string{"filter=active", "tag=inline"},
	}

	opts, cleanup, err := newHeaderTestService().BuildRequestOptions(cfg, "GET", "https://api.example.com/items")
	if cleanup != nil {
		defer cleanup()
	}

	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/items?filter=active&tag=inline", opts.URL)
}

func TestBuildRequestOptions_URLParamFileMissingReturnsError(t *testing.T) {
	cfg := config.Config{NoAuth: true, URLParamFile: filepath.Join(t.TempDir(), "nope.txt")}

	_, cleanup, err := newHeaderTestService().BuildRequestOptions(cfg, "GET", testHeaderURL)
	if cleanup != nil {
		defer cleanup()
	}

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open URL parameter file")
}

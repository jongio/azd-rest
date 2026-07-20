package service

import (
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestOptions_HeaderShortcutsSetHeaders(t *testing.T) {
	cfg := config.Config{
		NoAuth:      true,
		Accept:      "application/json",
		ContentType: "application/merge-patch+json",
	}

	opts, cleanup, err := newHeaderTestService().BuildRequestOptions(cfg, "PATCH", testHeaderURL)
	if cleanup != nil {
		defer cleanup()
	}

	require.NoError(t, err)
	assert.Equal(t, "application/json", opts.Headers["Accept"])
	assert.Equal(t, "application/merge-patch+json", opts.Headers[contentTypeHeader])
}

func TestBuildRequestOptions_HeaderShortcutsOverrideInlineHeaders(t *testing.T) {
	cfg := config.Config{
		NoAuth:      true,
		Headers:     []string{"Accept: application/xml", "Content-Type: application/json", "X-Trace: abc123"},
		Accept:      "application/json",
		ContentType: "application/json-patch+json",
	}

	opts, cleanup, err := newHeaderTestService().BuildRequestOptions(cfg, "PATCH", testHeaderURL)
	if cleanup != nil {
		defer cleanup()
	}

	require.NoError(t, err)
	assert.Equal(t, "application/json", opts.Headers["Accept"])
	assert.Equal(t, "application/json-patch+json", opts.Headers[contentTypeHeader])
	assert.Equal(t, "abc123", opts.Headers["X-Trace"])
}

func TestBuildRequestOptions_ContentTypeShortcutBlocksBodyDefault(t *testing.T) {
	cfg := config.Config{
		NoAuth:      true,
		JSONFields:  []string{"name=vm"},
		ContentType: "application/merge-patch+json",
	}

	opts, cleanup, err := newHeaderTestService().BuildRequestOptions(cfg, "PATCH", testHeaderURL)
	if cleanup != nil {
		defer cleanup()
	}

	require.NoError(t, err)
	assert.Equal(t, "application/merge-patch+json", opts.Headers[contentTypeHeader])
}

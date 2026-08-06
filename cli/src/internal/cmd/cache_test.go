package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jongio/azd-rest/src/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateCmdCacheDir points the user cache directory at a temp dir so the cache
// subcommand tests never touch a real user cache. It sets the variable
// os.UserCacheDir reads on each supported OS.
func isolateCmdCacheDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("LocalAppData", tmp)   // Windows
	t.Setenv("XDG_CACHE_HOME", tmp) // Linux
	t.Setenv("HOME", tmp)           // macOS and Unix fallback
}

func TestCachePathCommand(t *testing.T) {
	isolateCmdCacheDir(t)
	want, err := service.CacheDir()
	require.NoError(t, err)

	cmd := NewCacheCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"path"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, want, strings.TrimSpace(out.String()))
}

func TestCacheClearCommand(t *testing.T) {
	isolateCmdCacheDir(t)
	dir, err := service.CacheDir()
	require.NoError(t, err)

	cmd := NewCacheCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"clear"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "Cleared response cache")
	assert.Contains(t, out.String(), dir)
}

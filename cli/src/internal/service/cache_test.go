package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cacheExitCoder mirrors the cmd.ExitCoder contract so tests can assert the
// exit code an error carries without importing the cmd package.
type cacheExitCoder interface{ ExitCode() int }

// isolateCacheDir points the user cache directory at a temp dir for the test so
// cache reads and writes never touch a real user cache. It sets the variable
// os.UserCacheDir reads on each supported OS and returns the resolved cache dir.
func isolateCacheDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("LocalAppData", tmp)   // Windows
	t.Setenv("XDG_CACHE_HOME", tmp) // Linux
	t.Setenv("HOME", tmp)           // macOS ($HOME/Library/Caches) and Unix fallback
	dir, err := CacheDir()
	require.NoError(t, err)
	return dir
}

func TestParseCacheTTL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{"empty is off", "", 0, false},
		{"zero is off", "0", 0, false},
		{"whitespace is off", "   ", 0, false},
		{"seconds", "30s", 30 * time.Second, false},
		{"minutes", "5m", 5 * time.Minute, false},
		{"negative rejected", "-3s", 0, true},
		{"garbage rejected", "later", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCacheTTL(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				var coder cacheExitCoder
				require.True(t, errors.As(err, &coder), "error should carry an exit code")
				assert.Equal(t, 2, coder.ExitCode())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCacheKey(t *testing.T) {
	base := cacheKey("GET", "https://management.azure.com/subs?api-version=2020-01-01", "scope-a")

	t.Run("stable for identical input", func(t *testing.T) {
		assert.Equal(t, base, cacheKey("GET", "https://management.azure.com/subs?api-version=2020-01-01", "scope-a"))
	})
	t.Run("method is case-insensitive", func(t *testing.T) {
		assert.Equal(t, base, cacheKey("get", "https://management.azure.com/subs?api-version=2020-01-01", "scope-a"))
	})
	t.Run("scope changes the key", func(t *testing.T) {
		assert.NotEqual(t, base, cacheKey("GET", "https://management.azure.com/subs?api-version=2020-01-01", "scope-b"))
	})
	t.Run("url changes the key", func(t *testing.T) {
		assert.NotEqual(t, base, cacheKey("GET", "https://management.azure.com/other", "scope-a"))
	})
	t.Run("key is a json file name", func(t *testing.T) {
		assert.True(t, len(base) > 5 && base[len(base)-5:] == ".json")
	})
}

func TestWriteReadCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	key := cacheKey("GET", "https://example.com/data", "")
	want := &client.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"abc"}},
		Body:       []byte(`{"name":"kv"}`),
	}

	require.NoError(t, writeCache(dir, key, want))

	got, hit := readCache(dir, key, time.Minute)
	require.True(t, hit)
	assert.Equal(t, want.StatusCode, got.StatusCode)
	assert.Equal(t, want.Status, got.Status)
	assert.Equal(t, want.Body, got.Body)
	assert.Equal(t, "application/json", got.Headers.Get("Content-Type"))
	assert.Equal(t, "abc", got.Headers.Get("X-Request-Id"))
}

func TestReadCacheExpired(t *testing.T) {
	dir := t.TempDir()
	key := cacheKey("GET", "https://example.com/data", "")
	require.NoError(t, writeCache(dir, key, &client.Response{StatusCode: 200, Body: []byte(`{}`)}))

	// A zero TTL means every entry is already stale.
	_, hit := readCache(dir, key, 0)
	assert.False(t, hit)

	// A tiny TTL expires after a short sleep.
	time.Sleep(10 * time.Millisecond)
	_, hit = readCache(dir, key, time.Millisecond)
	assert.False(t, hit)
}

func TestReadCacheMissOnAbsentOrCorrupt(t *testing.T) {
	dir := t.TempDir()
	key := cacheKey("GET", "https://example.com/data", "")

	_, hit := readCache(dir, key, time.Minute)
	assert.False(t, hit, "absent entry is a miss")

	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, key), []byte("not json"), 0o600))
	_, hit = readCache(dir, key, time.Minute)
	assert.False(t, hit, "corrupt entry is a miss, not an error")
}

func TestWriteCacheFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permission bits do not apply on Windows")
	}
	dir := t.TempDir()
	key := cacheKey("GET", "https://example.com/data", "")
	require.NoError(t, writeCache(dir, key, &client.Response{StatusCode: 200, Body: []byte(`{}`)}))

	info, err := os.Stat(filepath.Join(dir, key))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestClearCache(t *testing.T) {
	dir := isolateCacheDir(t)
	key := cacheKey("GET", "https://example.com/data", "")
	require.NoError(t, writeCache(dir, key, &client.Response{StatusCode: 200, Body: []byte(`{}`)}))
	require.FileExists(t, filepath.Join(dir, key))

	cleared, err := ClearCache()
	require.NoError(t, err)
	assert.Equal(t, dir, cleared)
	assert.NoFileExists(t, filepath.Join(dir, key))

	// Clearing an already-empty cache is not an error.
	_, err = ClearCache()
	require.NoError(t, err)
}

func TestExecute_CacheTTL_ServesSecondFromCache(t *testing.T) {
	isolateCacheDir(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"hit":%d}`, n)
	}))
	defer srv.Close()

	run := func() string {
		tmp := filepath.Join(t.TempDir(), "out.json")
		cfg := config.Defaults()
		cfg.NoAuth = true
		cfg.OutputFile = tmp
		cfg.OutputFormat = "raw"
		cfg.CacheTTL = "5m"
		require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/items"))
		out, err := os.ReadFile(tmp)
		require.NoError(t, err)
		return string(out)
	}

	first := run()
	second := run()

	assert.Equal(t, int32(1), hits.Load(), "second identical GET should be served from cache")
	assert.Equal(t, first, second)
	assert.Contains(t, first, `"hit":1`)
}

func TestExecute_NoCache_ForcesFreshAndRefreshes(t *testing.T) {
	isolateCacheDir(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"hit":%d}`, n)
	}))
	defer srv.Close()

	run := func(noCache bool) string {
		tmp := filepath.Join(t.TempDir(), "out.json")
		cfg := config.Defaults()
		cfg.NoAuth = true
		cfg.OutputFile = tmp
		cfg.OutputFormat = "raw"
		cfg.CacheTTL = "5m"
		cfg.NoCache = noCache
		require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/items"))
		out, err := os.ReadFile(tmp)
		require.NoError(t, err)
		return string(out)
	}

	run(false) // warm the cache (hit 1)
	fresh := run(true)
	assert.Equal(t, int32(2), hits.Load(), "--no-cache must hit the network")
	assert.Contains(t, fresh, `"hit":2`)

	// The --no-cache call refreshed the entry, so the next cached read returns hit 2.
	cached := run(false)
	assert.Equal(t, int32(2), hits.Load(), "cached read after refresh should not hit the network")
	assert.Contains(t, cached, `"hit":2`)
}

func TestExecute_CacheTTL_ErrorStatusNotCached(t *testing.T) {
	isolateCacheDir(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"missing"}`))
	}))
	defer srv.Close()

	run := func() {
		tmp := filepath.Join(t.TempDir(), "out.json")
		cfg := config.Defaults()
		cfg.NoAuth = true
		cfg.OutputFile = tmp
		cfg.CacheTTL = "5m"
		require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/items"))
	}

	run()
	run()
	assert.Equal(t, int32(2), hits.Load(), "a non-2xx response must never be cached")
}

func TestExecute_CacheTTL_NonGetNotCached(t *testing.T) {
	isolateCacheDir(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	run := func() {
		tmp := filepath.Join(t.TempDir(), "out.json")
		cfg := config.Defaults()
		cfg.NoAuth = true
		cfg.OutputFile = tmp
		cfg.CacheTTL = "5m"
		require.NoError(t, newTestService().Execute(context.Background(), cfg, "POST", srv.URL+"/items"))
	}

	run()
	run()
	assert.Equal(t, int32(2), hits.Load(), "only GET responses are cached")
}

func TestExecute_InvalidCacheTTL_ExitsTwoBeforeRequest(t *testing.T) {
	isolateCacheDir(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.CacheTTL = "nope"

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/items")
	require.Error(t, err)
	var coder cacheExitCoder
	require.True(t, errors.As(err, &coder))
	assert.Equal(t, 2, coder.ExitCode())
	assert.Equal(t, int32(0), hits.Load(), "an invalid --cache-ttl must fail before any network call")
}

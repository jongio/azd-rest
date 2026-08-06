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
	"strings"
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

func testCacheOptions(method, requestURL, scope string) client.RequestOptions {
	return client.RequestOptions{
		Method:          method,
		URL:             requestURL,
		Scope:           scope,
		Headers:         map[string]string{},
		FollowRedirects: true,
		MaxRedirects:    10,
	}
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
	opts := testCacheOptions("GET", "https://management.azure.com/subs?api-version=2020-01-01", "scope-a")
	base := cacheKey(opts, "identity-a")

	t.Run("stable for identical input", func(t *testing.T) {
		assert.Equal(t, base, cacheKey(opts, "identity-a"))
	})
	t.Run("method is case-insensitive", func(t *testing.T) {
		changed := opts
		changed.Method = "get"
		assert.Equal(t, base, cacheKey(changed, "identity-a"))
	})
	t.Run("scope changes the key", func(t *testing.T) {
		changed := opts
		changed.Scope = "scope-b"
		assert.NotEqual(t, base, cacheKey(changed, "identity-a"))
	})
	t.Run("url changes the key", func(t *testing.T) {
		changed := opts
		changed.URL = "https://management.azure.com/other"
		assert.NotEqual(t, base, cacheKey(changed, "identity-a"))
	})
	t.Run("identity changes the key", func(t *testing.T) {
		assert.NotEqual(t, base, cacheKey(opts, "identity-b"))
	})
	t.Run("response headers change the key", func(t *testing.T) {
		changed := opts
		changed.Headers = map[string]string{"Accept": "application/xml"}
		assert.NotEqual(t, base, cacheKey(changed, "identity-a"))
	})
	t.Run("pagination changes the key", func(t *testing.T) {
		changed := opts
		changed.Paginate = true
		assert.NotEqual(t, base, cacheKey(changed, "identity-a"))
	})
	t.Run("redirect policy changes the key", func(t *testing.T) {
		changed := opts
		changed.FollowRedirects = false
		assert.NotEqual(t, base, cacheKey(changed, "identity-a"))
	})
	t.Run("TLS verification policy changes the key", func(t *testing.T) {
		changed := opts
		changed.Insecure = true
		assert.NotEqual(t, base, cacheKey(changed, "identity-a"))
	})
	t.Run("header names and order are normalized", func(t *testing.T) {
		changed := opts
		changed.Headers = map[string]string{
			"X-Mode": "full",
			"Accept": "application/json",
		}
		reordered := opts
		reordered.Headers = map[string]string{
			"accept": "application/json",
			"x-mode": "full",
		}
		assert.Equal(t, cacheKey(changed, "identity-a"), cacheKey(reordered, "identity-a"))
	})
	t.Run("key is a json file name", func(t *testing.T) {
		assert.True(t, len(base) > 5 && base[len(base)-5:] == ".json")
	})
}

func TestWriteReadCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	key := cacheKey(testCacheOptions("GET", "https://example.com/data", ""), "")
	want := &client.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"abc"}},
		Body:       []byte(`{"name":"kv"}`),
	}

	require.NoError(t, writeCache(dir, key, want))

	got, hit := readCache(dir, key, time.Minute, 1024)
	require.True(t, hit)
	assert.Equal(t, want.StatusCode, got.StatusCode)
	assert.Equal(t, want.Status, got.Status)
	assert.Equal(t, want.Body, got.Body)
	assert.Equal(t, "application/json", got.Headers.Get("Content-Type"))
	assert.Equal(t, "abc", got.Headers.Get("X-Request-Id"))
}

func TestReadCacheExpired(t *testing.T) {
	dir := t.TempDir()
	key := cacheKey(testCacheOptions("GET", "https://example.com/data", ""), "")
	require.NoError(t, writeCache(dir, key, &client.Response{StatusCode: 200, Body: []byte(`{}`)}))

	// A zero TTL means every entry is already stale.
	_, hit := readCache(dir, key, 0, 1024)
	assert.False(t, hit)
	assert.NoFileExists(t, filepath.Join(dir, key))

	// A tiny TTL expires after a short sleep.
	require.NoError(t, writeCache(dir, key, &client.Response{StatusCode: 200, Body: []byte(`{}`)}))
	time.Sleep(10 * time.Millisecond)
	_, hit = readCache(dir, key, time.Millisecond, 1024)
	assert.False(t, hit)
	assert.NoFileExists(t, filepath.Join(dir, key))
}

func TestReadCacheMissOnAbsentOrCorrupt(t *testing.T) {
	dir := t.TempDir()
	key := cacheKey(testCacheOptions("GET", "https://example.com/data", ""), "")

	_, hit := readCache(dir, key, time.Minute, 1024)
	assert.False(t, hit, "absent entry is a miss")

	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, key), []byte("not json"), 0o600))
	_, hit = readCache(dir, key, time.Minute, 1024)
	assert.False(t, hit, "corrupt entry is a miss, not an error")
	assert.NoFileExists(t, filepath.Join(dir, key))
}

func TestReadCacheRejectsOversizedBody(t *testing.T) {
	dir := t.TempDir()
	key := cacheKey(testCacheOptions("GET", "https://example.com/data", ""), "")
	require.NoError(t, writeCache(dir, key, &client.Response{StatusCode: 200, Body: []byte("oversized")}))

	_, hit := readCache(dir, key, time.Minute, 4)

	assert.False(t, hit)
	assert.NoFileExists(t, filepath.Join(dir, key))
}

func TestWriteCacheFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permission bits do not apply on Windows")
	}
	dir := t.TempDir()
	key := cacheKey(testCacheOptions("GET", "https://example.com/data", ""), "")
	require.NoError(t, writeCache(dir, key, &client.Response{StatusCode: 200, Body: []byte(`{}`)}))

	info, err := os.Stat(filepath.Join(dir, key))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestCacheRejectsSymlinkEntry(t *testing.T) {
	dir := t.TempDir()
	key := cacheKey(testCacheOptions("GET", "https://example.com/data", ""), "")
	target := filepath.Join(t.TempDir(), "target.json")
	require.NoError(t, os.WriteFile(target, []byte("unchanged"), 0o600))
	if err := os.Symlink(target, filepath.Join(dir, key)); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	_, hit := readCache(dir, key, time.Minute, 1024)
	assert.False(t, hit)
	require.NoError(t, os.Symlink(target, filepath.Join(dir, key)))
	require.Error(t, writeCache(dir, key, &client.Response{StatusCode: 200, Body: []byte(`{}`)}))

	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", string(data))
}

func TestCacheRejectsSymlinkDirectory(t *testing.T) {
	targetDir := t.TempDir()
	linkDir := filepath.Join(t.TempDir(), "cache-link")
	if err := os.Symlink(targetDir, linkDir); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	key := cacheKey(testCacheOptions("GET", "https://example.com/data", ""), "")
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, key), []byte(`{}`), 0o600))

	_, hit := readCache(linkDir, key, time.Minute, 1024)
	assert.False(t, hit)
	require.Error(t, removeCache(linkDir, key))
	require.Error(t, writeCache(linkDir, key, &client.Response{StatusCode: 200, Body: []byte(`{}`)}))
	assert.FileExists(t, filepath.Join(targetDir, key))
}

func TestPruneCacheBoundsEntryCount(t *testing.T) {
	dir := t.TempDir()
	oldestPath := ""
	keepPath := ""
	for i := 0; i <= cacheMaxEntries; i++ {
		path := filepath.Join(dir, fmt.Sprintf("%064x.json", i))
		require.NoError(t, os.WriteFile(path, []byte(`{}`), 0o600))
		if i == 0 {
			oldestPath = path
			old := time.Now().Add(-time.Hour)
			require.NoError(t, os.Chtimes(path, old, old))
		}
		keepPath = path
	}

	require.NoError(t, pruneCache(dir, keepPath))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, cacheMaxEntries)
	assert.NoFileExists(t, oldestPath)
	assert.FileExists(t, keepPath)
}

func TestClearCache(t *testing.T) {
	dir := isolateCacheDir(t)
	key := cacheKey(testCacheOptions("GET", "https://example.com/data", ""), "")
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

func TestNewCacheContextIsolatesCredentials(t *testing.T) {
	isolateCacheDir(t)
	first := testCacheOptions("GET", "https://management.azure.com/subscriptions", "scope")
	first.TokenProvider = &client.MockTokenProvider{Token: "token-a"}
	second := first
	second.TokenProvider = &client.MockTokenProvider{Token: "token-b"}

	firstContext, err := newCacheContext(context.Background(), &first)
	require.NoError(t, err)
	secondContext, err := newCacheContext(context.Background(), &second)
	require.NoError(t, err)

	assert.NotEqual(t, firstContext.key, secondContext.key)
	token, err := first.TokenProvider.GetToken(context.Background(), first.Scope)
	require.NoError(t, err)
	assert.Equal(t, "token-a", token, "request must use the credential that produced its cache key")
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

func TestExecute_CacheTTL_ValidatesAllowHostBeforeLookup(t *testing.T) {
	isolateCacheDir(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"secret":"cached"}`))
	}))
	defer srv.Close()

	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = filepath.Join(t.TempDir(), "out.json")
	cfg.CacheTTL = "5m"
	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))

	cfg.OutputFile = filepath.Join(t.TempDir(), "blocked.json")
	cfg.AllowedHosts = []string{"totally-different-host.example.com"}
	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not in the --allow-host allowlist")
	assert.Equal(t, int32(1), hits.Load())
}

func TestExecute_CacheTTL_SeparatesHeaderVariants(t *testing.T) {
	isolateCacheDir(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = fmt.Fprintf(w, `{"accept":%q}`, r.Header.Get("Accept"))
	}))
	defer srv.Close()

	run := func(accept string) {
		cfg := config.Defaults()
		cfg.NoAuth = true
		cfg.Accept = accept
		cfg.OutputFile = filepath.Join(t.TempDir(), "out.json")
		cfg.CacheTTL = "5m"
		require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))
	}

	run("application/json")
	run("application/xml")
	run("application/xml")

	assert.Equal(t, int32(2), hits.Load())
}

func TestExecute_CacheTTL_UsesDefaultResponseLimitWhenConfiguredZero(t *testing.T) {
	isolateCacheDir(t)
	var hits atomic.Int32
	body := strings.Repeat("x", 2*1024*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	run := func() {
		cfg := config.Defaults()
		cfg.NoAuth = true
		cfg.MaxResponseSize = 0
		cfg.OutputFile = filepath.Join(t.TempDir(), "out.txt")
		cfg.OutputFormat = "raw"
		cfg.CacheTTL = "5m"
		require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))
	}

	run()
	run()

	assert.Equal(t, int32(1), hits.Load())
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

func TestExecute_NoCache_RemovesStaleEntryAfterFailedRefresh(t *testing.T) {
	isolateCacheDir(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"allowed":true}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"allowed":false}`))
	}))
	defer srv.Close()

	run := func(noCache bool) {
		cfg := config.Defaults()
		cfg.NoAuth = true
		cfg.NoCache = noCache
		cfg.OutputFile = filepath.Join(t.TempDir(), "out.json")
		cfg.CacheTTL = "5m"
		require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))
	}

	run(false)
	run(true)
	run(false)

	assert.Equal(t, int32(3), hits.Load(), "failed refresh must not leave the old successful entry")
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

package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
)

// cacheContext holds everything derived from a request that the cache needs:
// the directory to read and write, the file key, and the final URL used for
// write-out expansion on a cache hit.
type cacheContext struct {
	dir      string
	key      string
	finalURL string
}

// newCacheContext resolves the final URL and scope the same way the request
// builder does, then derives the cache directory and key. It mirrors
// BuildRequestOptions so a cache hit keys off the exact request that would have
// been sent. Scope detection here never acquires a token.
func newCacheContext(cfg config.Config, method, rawURL string) (cacheContext, error) {
	finalURL, err := applyAPIVersion(rawURL, cfg.APIVersion)
	if err != nil {
		return cacheContext{}, err
	}
	finalURL, err = applyURLParams(finalURL, cfg.URLParams)
	if err != nil {
		return cacheContext{}, err
	}

	scope := cfg.Scope
	if scope == "" && !cfg.NoAuth {
		if detected, detectErr := auth.DetectScope(finalURL); detectErr == nil {
			scope = detected
		}
	}

	dir, err := CacheDir()
	if err != nil {
		return cacheContext{}, err
	}
	return cacheContext{dir: dir, key: cacheKey(method, finalURL, scope), finalURL: finalURL}, nil
}

// cacheConfigError signals that --cache-ttl was given an invalid duration. It
// reports exit code 2 (the invalid-configuration code) through the ExitCoder
// contract so main can distinguish it from a request failure.
type cacheConfigError struct{ err error }

func (e *cacheConfigError) Error() string { return e.err.Error() }

// ExitCode returns 2 for an invalid --cache-ttl value.
func (e *cacheConfigError) ExitCode() int { return 2 }

// cacheEnvelope is the on-disk representation of a cached response. Only the
// fields needed to reconstruct output are stored; timing is intentionally
// dropped because it describes the original request, not the cached read.
type cacheEnvelope struct {
	StatusCode int         `json:"status_code"`
	Status     string      `json:"status"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body"`
}

// parseCacheTTL interprets the raw --cache-ttl value. An empty value or "0"
// means caching is off. Any other value must be a positive Go duration
// (for example 30s, 5m, 1h). A malformed or negative value is a configuration
// error that exits with code 2.
func parseCacheTTL(raw string) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "0" {
		return 0, nil
	}
	ttl, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, &cacheConfigError{fmt.Errorf("invalid --cache-ttl %q: %w", raw, err)}
	}
	if ttl < 0 {
		return 0, &cacheConfigError{fmt.Errorf("invalid --cache-ttl %q: duration must not be negative", raw)}
	}
	return ttl, nil
}

// CacheDir returns the directory that holds cached responses. It lives under
// the user cache directory so it does not clutter the working tree and is
// scoped per user.
func CacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user cache directory: %w", err)
	}
	return filepath.Join(base, "azd-rest", "cache"), nil
}

// ClearCache removes every cached response. It returns the directory that was
// cleared so callers can report it. Removing a directory that does not exist is
// not an error.
func ClearCache() (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("failed to clear cache at %s: %w", dir, err)
	}
	return dir, nil
}

// cacheKey derives a stable file name for a request from its method, final URL,
// and scope. Including the scope keeps responses for different audiences from
// colliding even when the URL is identical.
func cacheKey(method, finalURL, scope string) string {
	sum := sha256.Sum256([]byte(strings.ToUpper(method) + "\n" + finalURL + "\n" + scope))
	return hex.EncodeToString(sum[:]) + ".json"
}

// readCache returns the cached response for key when a fresh entry exists. A
// missing file or an entry older than ttl reports ok=false so the caller falls
// back to the network. A corrupt entry is treated as a miss rather than an
// error so a bad file never blocks a request.
func readCache(dir, key string, ttl time.Duration) (*client.Response, bool) {
	path := filepath.Join(dir, key)
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if ttl <= 0 || time.Since(info.ModTime()) > ttl {
		return nil, false
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path is a sha256 hex key under the app cache dir.
	if err != nil {
		return nil, false
	}
	var env cacheEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, false
	}
	return &client.Response{
		StatusCode: env.StatusCode,
		Status:     env.Status,
		Headers:    env.Headers,
		Body:       env.Body,
	}, true
}

// writeCache stores resp under key with owner-only permissions. Cached bodies
// can hold sensitive data, so the directory is 0700 and the file is 0600. A
// write failure is returned so the caller can surface a note without failing
// the request.
func writeCache(dir, key string, resp *client.Response) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	env := cacheEnvelope{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Headers,
		Body:       resp.Body,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to encode cache entry: %w", err)
	}
	path := filepath.Join(dir, key)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write cache entry: %w", err)
	}
	return nil
}

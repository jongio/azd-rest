package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
)

const (
	cacheEnvelopeOverhead = int64(1024 * 1024)
	cacheMaxEntries       = 256
	cacheMaxBytes         = int64(512 * 1024 * 1024)
)

// cacheContext holds the directory and request-specific key for a cache entry.
type cacheContext struct {
	dir string
	key string
}

// staticTokenProvider preserves the exact credential used to derive an
// authenticated cache key, so the eventual network request cannot switch
// identities between key generation and execution.
type staticTokenProvider struct {
	token string
}

// GetToken returns the credential captured during cache-key generation.
func (p staticTokenProvider) GetToken(context.Context, string) (string, error) {
	return p.token, nil
}

// newCacheContext derives a cache key from fully validated request options.
// Authenticated requests acquire a token and fingerprint it before lookup so
// entries can never cross credential boundaries.
func newCacheContext(ctx context.Context, opts *client.RequestOptions) (cacheContext, error) {
	identity := ""
	if !opts.SkipAuth {
		if opts.TokenProvider == nil {
			return cacheContext{}, fmt.Errorf("cannot cache authenticated request without a token provider")
		}
		token, err := opts.TokenProvider.GetToken(ctx, opts.Scope)
		if err != nil {
			return cacheContext{}, fmt.Errorf("failed to acquire token for response cache: %w", err)
		}
		identity = credentialFingerprint(token)
		opts.TokenProvider = staticTokenProvider{token: token}
	}

	dir, err := CacheDir()
	if err != nil {
		return cacheContext{}, err
	}
	return cacheContext{dir: dir, key: cacheKey(*opts, identity)}, nil
}

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
// (for example 30s, 5m, 1h). A malformed or negative value is a structured
// configuration error.
func parseCacheTTL(raw string) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "0" {
		return 0, nil
	}
	ttl, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, newCacheConfigError(fmt.Sprintf("invalid --cache-ttl %q: %v", raw, err))
	}
	if ttl < 0 {
		return 0, newCacheConfigError(fmt.Sprintf("invalid --cache-ttl %q: duration must not be negative", raw))
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

// cacheKey derives a stable file name from every request input that can change
// the response representation.
func cacheKey(opts client.RequestOptions, identity string) string {
	headers := make([]string, 0, len(opts.Headers))
	for name, value := range opts.Headers {
		normalizedName := strings.ToLower(strings.TrimSpace(name))
		headers = append(headers, normalizedName+"\x00"+value)
	}
	sort.Strings(headers)

	material := struct {
		Method          string
		URL             string
		Scope           string
		Identity        string
		Headers         []string
		Paginate        bool
		FollowRedirects bool
		MaxRedirects    int
		Insecure        bool
	}{
		Method:          strings.ToUpper(opts.Method),
		URL:             opts.URL,
		Scope:           opts.Scope,
		Identity:        identity,
		Headers:         headers,
		Paginate:        opts.Paginate,
		FollowRedirects: opts.FollowRedirects,
		MaxRedirects:    opts.MaxRedirects,
		Insecure:        opts.Insecure,
	}
	encoded, err := json.Marshal(material)
	if err != nil {
		panic(fmt.Sprintf("failed to encode cache key material: %v", err))
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]) + ".json"
}

func credentialFingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// readCache returns the cached response for key when a fresh entry exists. A
// missing file or an entry older than ttl reports ok=false so the caller falls
// back to the network. A corrupt entry is treated as a miss rather than an
// error so a bad file never blocks a request.
func readCache(dir, key string, ttl time.Duration, maxResponseSize int64) (*client.Response, bool) {
	if err := validateCacheDir(dir); err != nil {
		return nil, false
	}
	path := filepath.Join(dir, key)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, false
	}
	if !info.Mode().IsRegular() {
		_ = os.Remove(path)
		return nil, false
	}
	if ttl <= 0 || time.Since(info.ModTime()) > ttl {
		_ = os.Remove(path)
		return nil, false
	}

	file, err := os.Open(path) // #nosec G304 -- path is a sha256 hex key under the app cache dir.
	if err != nil {
		return nil, false
	}
	openedInfo, err := file.Stat()
	if err != nil || !os.SameFile(info, openedInfo) {
		_ = file.Close()
		return nil, false
	}

	readLimit := cacheReadLimit(maxResponseSize)
	data, err := io.ReadAll(io.LimitReader(file, readLimit+1))
	closeErr := file.Close()
	if err != nil || closeErr != nil || int64(len(data)) > readLimit {
		_ = os.Remove(path)
		return nil, false
	}
	var env cacheEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		_ = os.Remove(path)
		return nil, false
	}
	if maxResponseSize > 0 && int64(len(env.Body)) > maxResponseSize {
		_ = os.Remove(path)
		return nil, false
	}
	return &client.Response{
		StatusCode: env.StatusCode,
		Status:     env.Status,
		Headers:    env.Headers,
		Body:       env.Body,
	}, true
}

func cacheReadLimit(maxResponseSize int64) int64 {
	if maxResponseSize <= 0 {
		return cacheEnvelopeOverhead
	}
	if maxResponseSize > (math.MaxInt64-cacheEnvelopeOverhead)/2 {
		return math.MaxInt64
	}
	return maxResponseSize*2 + cacheEnvelopeOverhead
}

// writeCache stores resp via atomic replacement. Permission bits restrict the
// directory and entry on platforms that support them.
func writeCache(dir, key string, resp *client.Response) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := validateCacheDir(dir); err != nil {
		return err
	}
	// #nosec G302 -- 0o700 is correct for a directory; it needs the execute bit to be traversable.
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("failed to protect cache directory: %w", err)
	}
	env := cacheEnvelope{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Headers,
		Body:       resp.Body,
	}

	path := filepath.Join(dir, key)
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to replace non-regular cache entry")
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect cache entry: %w", err)
	}

	temp, err := os.CreateTemp(dir, ".cache-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary cache entry: %w", err)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("failed to protect temporary cache entry: %w", err)
	}
	if err := json.NewEncoder(temp).Encode(env); err != nil {
		_ = temp.Close()
		return fmt.Errorf("failed to encode cache entry: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("failed to flush cache entry: %w", err)
	}
	tempInfo, err := temp.Stat()
	if err != nil {
		_ = temp.Close()
		return fmt.Errorf("failed to inspect temporary cache entry: %w", err)
	}
	if tempInfo.Size() > cacheMaxBytes {
		_ = temp.Close()
		return fmt.Errorf("cache entry exceeds the %d-byte cache limit", cacheMaxBytes)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("failed to close cache entry: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("failed to replace cache entry: %w", err)
		}
		if retryErr := os.Rename(tempPath, path); retryErr != nil {
			return fmt.Errorf("failed to replace cache entry: %w", retryErr)
		}
	}
	if err := pruneCache(dir, path); err != nil {
		return fmt.Errorf("failed to prune response cache: %w", err)
	}
	return nil
}

func removeCache(dir, key string) error {
	if err := validateCacheDir(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	path := filepath.Join(dir, key)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache entry: %w", err)
	}
	return nil
}

func validateCacheDir(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to use non-directory cache path")
	}
	return nil
}

type cacheFileInfo struct {
	path    string
	size    int64
	modTime time.Time
}

func pruneCache(dir, keepPath string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	files := make([]cacheFileInfo, 0, len(entries))
	var totalBytes int64
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		file := cacheFileInfo{
			path:    filepath.Join(dir, entry.Name()),
			size:    info.Size(),
			modTime: info.ModTime(),
		}
		files = append(files, file)
		totalBytes += file.size
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})
	remaining := len(files)
	for _, file := range files {
		if remaining <= cacheMaxEntries && totalBytes <= cacheMaxBytes {
			break
		}
		if file.path == keepPath && file.size <= cacheMaxBytes {
			continue
		}
		if err := os.Remove(file.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		totalBytes -= file.size
		remaining--
	}
	return nil
}

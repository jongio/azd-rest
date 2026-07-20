package service

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteDiagnostic_NotSilent verifies advisory messages are written when
// silent mode is off.
func TestWriteDiagnostic_NotSilent(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostic(&buf, false, "Warning: %s\n", "disabled")

	got := buf.String()
	if !strings.Contains(got, "Warning: disabled") {
		t.Fatalf("expected diagnostic to be written, got %q", got)
	}
}

// TestWriteDiagnostic_Silent verifies advisory messages are suppressed when
// silent mode is on.
func TestWriteDiagnostic_Silent(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostic(&buf, true, "Warning: %s\n", "disabled")

	if got := buf.String(); got != "" {
		t.Fatalf("expected no diagnostic output in silent mode, got %q", got)
	}
}

// TestWriteDiagnostic_FormatsArgs verifies the helper formats arguments like
// fmt.Fprintf when not silent.
func TestWriteDiagnostic_FormatsArgs(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostic(&buf, false, "> Pagination enabled (max %d pages)\n", 100)

	if got := buf.String(); got != "> Pagination enabled (max 100 pages)\n" {
		t.Fatalf("unexpected formatted output: %q", got)
	}
}

// newTestService builds a RequestService whose token provider is never used
// (tests target http:// endpoints where authentication is skipped).
func newTestService() *RequestService {
	return NewRequestService(
		func() (client.TokenProvider, error) { return nil, nil },
		DefaultHTTPClientFactory,
	)
}

func baseTestConfig(t *testing.T) config.Config {
	t.Helper()
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = filepath.Join(t.TempDir(), "out.json")
	return cfg
}

func TestBuildResponseHeaderBlock_StatusAndSortedHeaders(t *testing.T) {
	resp := &client.Response{
		Status: "200 OK",
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"abc-123"},
			"Etag":         []string{`"v1"`},
		},
	}

	block := buildResponseHeaderBlock(resp)
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")

	require.Equal(t, "200 OK", lines[0])
	// Headers are sorted alphabetically after the status line.
	assert.Equal(t, "Content-Type: application/json", lines[1])
	assert.Equal(t, `Etag: "v1"`, lines[2])
	assert.Equal(t, "X-Request-Id: abc-123", lines[3])
	// Block ends with a blank line separating headers from the body.
	assert.True(t, strings.HasSuffix(block, "\n\n"))
}

func TestBuildResponseHeaderBlock_RedactsSensitiveHeaders(t *testing.T) {
	resp := &client.Response{
		Status: "200 OK",
		Headers: http.Header{
			"Authorization": []string{"******"},
			"Set-Cookie":    []string{"session=super-secret-cookie-value"},
		},
	}

	block := buildResponseHeaderBlock(resp)

	assert.NotContains(t, block, "super-secret-token-value")
	assert.NotContains(t, block, "super-secret-cookie-value")
}

func TestExecute_Include_JSONPrependsHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-42")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.txt")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.Include = true
	cfg.OutputFile = tmp

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/api")
	require.NoError(t, err)

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	body := string(out)

	assert.Contains(t, body, "200 OK")
	assert.Contains(t, body, "X-Request-Id: req-42")
	assert.Contains(t, body, `"result": "ok"`)
	// Header block precedes the body.
	assert.Less(t, strings.Index(body, "X-Request-Id"), strings.Index(body, "result"))
}

func TestExecute_NoInclude_BodyOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-99")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.txt")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.Include = false
	cfg.OutputFile = tmp

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/api")
	require.NoError(t, err)

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	body := string(out)

	assert.NotContains(t, body, "X-Request-Id")
	assert.NotContains(t, body, "200 OK")
	assert.Contains(t, body, `"result": "ok"`)
}

func TestExecute_Include_BinaryPrependsHeaders(t *testing.T) {
	payload := []byte{0x00, 0x01, 0x02, 0x03, 0xff}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Request-Id", "bin-7")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.bin")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.Include = true
	cfg.Binary = true
	cfg.OutputFile = tmp

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/blob")
	require.NoError(t, err)

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(string(out), "200 OK"))
	assert.Contains(t, string(out), "X-Request-Id: bin-7")
	// Raw binary payload is preserved after the header block.
	assert.True(t, strings.HasSuffix(string(out), string(payload)))
}

func TestExecute_MaxTimeExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := baseTestConfig(t)
	cfg.Retry = 0
	cfg.MaxTime = 20 * time.Millisecond

	err := newTestService().Execute(context.Background(), cfg, "GET", server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--max-time")
}

func TestExecute_WithinMaxTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := baseTestConfig(t)
	cfg.MaxTime = 5 * time.Second

	err := newTestService().Execute(context.Background(), cfg, "GET", server.URL)
	require.NoError(t, err)
}

func TestExecute_MaxTimeDisabledByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := baseTestConfig(t)
	cfg.MaxTime = 0

	err := newTestService().Execute(context.Background(), cfg, "GET", server.URL)
	require.NoError(t, err)
}

func TestExecute_ReadOnlyAllowsSafeMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	for _, method := range []string{"GET", "HEAD", "OPTIONS"} {
		t.Run(method, func(t *testing.T) {
			cfg := baseTestConfig(t)
			cfg.ReadOnly = true

			err := newTestService().Execute(context.Background(), cfg, method, server.URL)
			require.NoError(t, err)
		})
	}
}

func TestExecute_ReadOnlyBlocksMutatingBeforeDependencies(t *testing.T) {
	tokenProviderCalls := 0
	httpClientCalls := 0
	svc := NewRequestService(
		func() (client.TokenProvider, error) {
			tokenProviderCalls++
			return nil, nil
		},
		func(tp client.TokenProvider, insecure bool, timeout time.Duration) *client.Client {
			httpClientCalls++
			return nil
		},
	)

	cfg := config.Defaults()
	cfg.ReadOnly = true

	err := svc.Execute(context.Background(), cfg, "POST", "https://management.azure.com/subscriptions?api-version=2021-04-01")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--read-only blocks POST requests")
	assert.Zero(t, tokenProviderCalls, "read-only validation should run before token creation")
	assert.Zero(t, httpClientCalls, "read-only validation should run before client creation")
}

func TestBuildRequestOptions_AllowHostPermitsMatch(t *testing.T) {
	svc := newTestService()
	cfg := baseTestConfig(t)
	cfg.AllowedHosts = []string{"management.azure.com"}

	opts, cleanup, err := svc.BuildRequestOptions(cfg, "GET", "https://management.azure.com/subscriptions?api-version=2021-04-01")
	if cleanup != nil {
		cleanup()
	}
	require.NoError(t, err)
	assert.Equal(t, "https://management.azure.com/subscriptions?api-version=2021-04-01", opts.URL)
}

func TestBuildRequestOptions_AllowHostRejectsMismatch(t *testing.T) {
	svc := newTestService()
	cfg := baseTestConfig(t)
	cfg.AllowedHosts = []string{"management.azure.com"}

	_, cleanup, err := svc.BuildRequestOptions(cfg, "GET", "https://evil.example.com/data")
	if cleanup != nil {
		cleanup()
	}
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evil.example.com")
	assert.Contains(t, err.Error(), "allow-host")
}

func TestBuildRequestOptions_AllowHostIgnoresPort(t *testing.T) {
	svc := newTestService()
	cfg := baseTestConfig(t)
	cfg.AllowedHosts = []string{"localhost"}

	_, cleanup, err := svc.BuildRequestOptions(cfg, "GET", "https://localhost:8443/probe")
	if cleanup != nil {
		cleanup()
	}
	require.NoError(t, err)
}

func TestBuildRequestOptions_AllowHostWildcard(t *testing.T) {
	svc := newTestService()
	cfg := baseTestConfig(t)
	cfg.AllowedHosts = []string{"*.vault.azure.net"}

	_, cleanup, err := svc.BuildRequestOptions(cfg, "GET", "https://kv.vault.azure.net/secrets/x")
	if cleanup != nil {
		cleanup()
	}
	require.NoError(t, err)
}

func TestBuildRequestOptions_AllowHostUnsetAllowsAny(t *testing.T) {
	svc := newTestService()
	cfg := baseTestConfig(t)

	_, cleanup, err := svc.BuildRequestOptions(cfg, "GET", "https://anything.example.com/x")
	if cleanup != nil {
		cleanup()
	}
	require.NoError(t, err)
}

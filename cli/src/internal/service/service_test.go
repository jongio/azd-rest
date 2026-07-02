package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestService returns a RequestService whose token provider is never used
// (tests set NoAuth) and which uses the real HTTP client against httptest.
func newTestService() *RequestService {
	return NewRequestService(
		func() (client.TokenProvider, error) {
			return &client.MockTokenProvider{Token: "test-token"}, nil
		},
		DefaultHTTPClientFactory,
	)
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
			"Authorization": []string{"Bearer super-secret-token-value"},
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

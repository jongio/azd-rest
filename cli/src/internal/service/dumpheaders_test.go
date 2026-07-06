package service

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-rest/src/internal/client"
)

func newDumpHeadersResponse() *client.Response {
	return &client.Response{
		Status: "200 OK",
		Headers: http.Header{
			"Content-Type":  []string{"application/json"},
			"X-Request-Id":  []string{"req-99"},
			"Authorization": []string{"Bearer super-secret-token-value"},
		},
	}
}

func TestDumpResponseHeaders_ToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "headers.txt")
	if err := dumpResponseHeaders(path, newDumpHeadersResponse()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path) // #nosec G304 -- test-controlled temp path
	if err != nil {
		t.Fatalf("reading dumped headers: %v", err)
	}
	got := string(data)
	if !strings.HasPrefix(got, "200 OK\n") {
		t.Errorf("expected status line first, got:\n%s", got)
	}
	for _, want := range []string{"Content-Type: application/json", "X-Request-Id: req-99"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestDumpResponseHeaders_RedactsSensitive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "headers.txt")
	if err := dumpResponseHeaders(path, newDumpHeadersResponse()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path) // #nosec G304 -- test-controlled temp path
	if err != nil {
		t.Fatalf("reading dumped headers: %v", err)
	}
	if strings.Contains(string(data), "super-secret-token-value") {
		t.Errorf("sensitive header value was not redacted:\n%s", data)
	}
}

func TestDumpResponseHeaders_ToStderr(t *testing.T) {
	old := os.Stderr
	f, err := os.CreateTemp(t.TempDir(), "stderr-*.txt")
	if err != nil {
		t.Fatalf("creating temp stderr: %v", err)
	}
	os.Stderr = f
	dumpErr := dumpResponseHeaders("-", newDumpHeadersResponse())
	os.Stderr = old
	_ = f.Close()

	if dumpErr != nil {
		t.Fatalf("unexpected error: %v", dumpErr)
	}

	data, err := os.ReadFile(f.Name()) // #nosec G304 -- test-controlled temp path
	if err != nil {
		t.Fatalf("reading captured stderr: %v", err)
	}
	if !strings.Contains(string(data), "200 OK") {
		t.Errorf("expected header block on stderr, got:\n%s", data)
	}
}

func TestDumpResponseHeaders_WriteError(t *testing.T) {
	// Parent directory does not exist, so the write must fail.
	path := filepath.Join(t.TempDir(), "missing-dir", "headers.txt")
	if err := dumpResponseHeaders(path, newDumpHeadersResponse()); err == nil {
		t.Fatal("expected an error writing to a missing directory")
	}
}

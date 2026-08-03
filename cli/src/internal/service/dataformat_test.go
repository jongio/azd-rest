package service

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
)

// dataFormatExitCoder is a local structural interface used to assert the exit
// code carried by --data-format failures without importing the cmd package.
type dataFormatExitCoder interface{ ExitCode() int }

func TestYamlToJSON(t *testing.T) {
	in := []byte("name: alpha\ncount: 3\nnested:\n  enabled: true\ntags:\n  - a\n  - b\n")
	got, err := yamlToJSON(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"count":3,"name":"alpha","nested":{"enabled":true},"tags":["a","b"]}`
	if string(got) != want {
		t.Fatalf("yamlToJSON = %q, want %q", string(got), want)
	}
}

func TestYamlToJSONAcceptsJSONInput(t *testing.T) {
	in := []byte(`{"a":1,"b":[2,3]}`)
	got, err := yamlToJSON(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"a":1,"b":[2,3]}`
	if string(got) != want {
		t.Fatalf("yamlToJSON = %q, want %q", string(got), want)
	}
}

func TestBuildRequestOptions_DataFormatYAMLInline(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		Data:       "name: alpha\ncount: 3\n",
		DataFormat: dataFormatYAML,
	}

	opts, cleanup, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if opts.Headers[contentTypeHeader] != applicationJSON {
		t.Errorf("Content-Type = %q, want %q", opts.Headers[contentTypeHeader], applicationJSON)
	}
	body, err := io.ReadAll(opts.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	want := `{"count":3,"name":"alpha"}`
	if string(body) != want {
		t.Errorf("body = %q, want %q", string(body), want)
	}
}

func TestBuildRequestOptions_DataFormatYAMLFile(t *testing.T) {
	svc := newTestService()
	path := filepath.Join(t.TempDir(), "body.yaml")
	if err := os.WriteFile(path, []byte("region: eastus\nenabled: true\n"), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	cfg := config.Config{
		NoAuth:     true,
		DataFile:   path,
		DataFormat: dataFormatYAML,
	}

	opts, cleanup, err := svc.BuildRequestOptions(cfg, "PUT", "https://example.com/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	body, err := io.ReadAll(opts.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	want := `{"enabled":true,"region":"eastus"}`
	if string(body) != want {
		t.Errorf("body = %q, want %q", string(body), want)
	}
}

func TestBuildRequestOptions_DataFormatYAMLKeepsUserContentType(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		Headers:    []string{"Content-Type: application/merge-patch+json"},
		Data:       "a: 1\n",
		DataFormat: dataFormatYAML,
	}

	opts, cleanup, err := svc.BuildRequestOptions(cfg, "PATCH", "https://example.com/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if opts.Headers[contentTypeHeader] != "application/merge-patch+json" {
		t.Errorf("Content-Type = %q, want application/merge-patch+json", opts.Headers[contentTypeHeader])
	}
}

func TestBuildRequestOptions_DataFormatJSONUnchanged(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		Data:       `{"a":1}`,
		DataFormat: dataFormatJSON,
	}

	opts, cleanup, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	// Default JSON path is a raw passthrough that does not force a Content-Type.
	if _, ok := opts.Headers[contentTypeHeader]; ok {
		t.Errorf("Content-Type should not be set by default JSON path, got %q", opts.Headers[contentTypeHeader])
	}
	body, err := io.ReadAll(opts.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if string(body) != `{"a":1}` {
		t.Errorf("body = %q, want %q", string(body), `{"a":1}`)
	}
}

func TestBuildRequestOptions_DataFormatInvalidValue(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		Data:       "a: 1\n",
		DataFormat: "toml",
	}

	_, _, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x")
	assertDataFormatExit2(t, err)
}

func TestBuildRequestOptions_DataFormatInvalidYAML(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		Data:       "key: [1, 2",
		DataFormat: dataFormatYAML,
	}

	_, _, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x")
	assertDataFormatExit2(t, err)
}

func TestBuildRequestOptions_DataFormatYAMLConflicts(t *testing.T) {
	svc := newTestService()
	conflicts := []config.Config{
		{NoAuth: true, DataFormat: dataFormatYAML, Data: "a: 1\n", JSONFields: []string{"a=1"}},
		{NoAuth: true, DataFormat: dataFormatYAML, Data: "a: 1\n", JSONFieldsRaw: []string{"a:=1"}},
		{NoAuth: true, DataFormat: dataFormatYAML, Data: "a: 1\n", FormFields: []string{"b=2"}},
	}
	for _, cfg := range conflicts {
		_, _, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x")
		assertDataFormatExit2(t, err)
	}
}

func assertDataFormatExit2(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var coder dataFormatExitCoder
	if !errors.As(err, &coder) {
		t.Fatalf("error %v does not carry an exit code", err)
	}
	if coder.ExitCode() != 2 {
		t.Fatalf("exit code = %d, want 2", coder.ExitCode())
	}
}

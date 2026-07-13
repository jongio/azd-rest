package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// templateCoder mirrors the structural exit-code contract used by main so the
// tests can assert the exit code without importing the cmd package.
type templateCoder interface{ ExitCode() int }

func TestRenderTemplate_ARMValueNames(t *testing.T) {
	body := []byte(`{"value":[{"name":"a"},{"name":"b"},{"name":"c"}]}`)
	out, err := renderTemplate(`{{range .value}}{{.name}}{{"\n"}}{{end}}`, body)
	require.NoError(t, err)
	assert.Equal(t, "a\nb\nc\n", out)
}

func TestRenderTemplate_Helpers(t *testing.T) {
	body := []byte(`{"name":"Kv","tags":["x","y"],"nested":{"k":1}}`)

	tests := []struct {
		name string
		tmpl string
		want string
	}{
		{name: "upper", tmpl: `{{upper .name}}`, want: "KV"},
		{name: "lower", tmpl: `{{lower .name}}`, want: "kv"},
		{name: "join", tmpl: `{{join "," .tags}}`, want: "x,y"},
		{name: "json", tmpl: `{{json .nested}}`, want: `{"k":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := renderTemplate(tt.tmpl, body)
			require.NoError(t, err)
			assert.Equal(t, tt.want, out)
		})
	}
}

func TestRenderTemplate_InvalidSyntaxExits2(t *testing.T) {
	_, err := renderTemplate(`{{range .value}}`, []byte(`{}`))
	require.Error(t, err)

	var coder templateCoder
	require.True(t, errors.As(err, &coder), "invalid syntax should carry an exit code")
	assert.Equal(t, 2, coder.ExitCode())
}

func TestRenderTemplate_NonJSONReportsError(t *testing.T) {
	_, err := renderTemplate(`{{.name}}`, []byte("not json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON")
}

func TestParseTemplate_MissingFileExits2(t *testing.T) {
	_, err := parseTemplate("@" + filepath.Join(t.TempDir(), "missing.tmpl"))
	require.Error(t, err)

	var coder templateCoder
	require.True(t, errors.As(err, &coder), "missing file should carry an exit code")
	assert.Equal(t, 2, coder.ExitCode())
}

func TestRenderTemplate_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.tmpl")
	require.NoError(t, os.WriteFile(path, []byte(`id={{.id}}`), 0o600))

	out, err := renderTemplate("@"+path, []byte(`{"id":"abc"}`))
	require.NoError(t, err)
	assert.Equal(t, "id=abc", out)
}

func TestExecute_Template_TakesPrecedenceOverFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[{"name":"one"},{"name":"two"}]}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.txt")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.OutputFormat = "yaml"
	cfg.Template = `{{range .value}}{{.name}} {{end}}`

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/list")
	require.NoError(t, err)

	out, readErr := os.ReadFile(tmp)
	require.NoError(t, readErr)
	got := string(out)
	assert.Equal(t, "one two ", got)
	assert.NotContains(t, strings.ToLower(got), "name") // structured format keys not emitted
}

func TestExecute_Template_RunsAfterQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[{"name":"a"},{"name":"b"}]}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.txt")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.Query = "value"
	cfg.Template = `{{range .}}{{.name}}-{{end}}`

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/list")
	require.NoError(t, err)

	out, readErr := os.ReadFile(tmp)
	require.NoError(t, readErr)
	assert.Equal(t, "a-b-", string(out))
}

func TestExecute_Template_InvalidExitsBeforeRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = filepath.Join(t.TempDir(), "out.txt")
	cfg.Template = `{{range .value}}`

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/never")
	require.Error(t, err)

	var coder templateCoder
	require.True(t, errors.As(err, &coder))
	assert.Equal(t, 2, coder.ExitCode())
	assert.False(t, called, "no request should be made when the template is invalid")
}

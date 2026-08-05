package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectFields_Object(t *testing.T) {
	body := []byte(`{"name":"kv","location":"eastus","id":"/subs/x","type":"vault"}`)
	out, ok := projectFields(body, []string{"name", "location"})
	require.True(t, ok)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	assert.Equal(t, map[string]any{"name": "kv", "location": "eastus"}, got)
}

func TestProjectFields_ArrayOfObjects(t *testing.T) {
	body := []byte(`[{"name":"a","id":1},{"name":"b","id":2}]`)
	out, ok := projectFields(body, []string{"name"})
	require.True(t, ok)

	var got []map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	assert.Equal(t, []map[string]any{{"name": "a"}, {"name": "b"}}, got)
}

func TestProjectFields_ARMValueWrapperKeepsPagingLink(t *testing.T) {
	body := []byte(`{"value":[{"name":"a","id":"/x/a"},{"name":"b","id":"/x/b"}],"nextLink":"https://next"}`)
	out, ok := projectFields(body, []string{"name"})
	require.True(t, ok)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	assert.Equal(t, "https://next", got["nextLink"])

	values, isArray := got["value"].([]any)
	require.True(t, isArray)
	require.Len(t, values, 2)
	first := values[0].(map[string]any)
	assert.Equal(t, map[string]any{"name": "a"}, first)
}

func TestProjectFields_MissingFieldOmitted(t *testing.T) {
	body := []byte(`{"name":"kv"}`)
	out, ok := projectFields(body, []string{"name", "missing"})
	require.True(t, ok)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	assert.Equal(t, map[string]any{"name": "kv"}, got)
}

func TestProjectFields_NonJSONReturnsFalse(t *testing.T) {
	body := []byte("not json")
	out, ok := projectFields(body, []string{"name"})
	assert.False(t, ok)
	assert.Equal(t, body, out)
}

func TestExecute_Fields_TrimsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"kv","location":"eastus","id":"/subs/x"}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.json")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.Fields = []string{"name", "location"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/x")
	require.NoError(t, err)

	out, readErr := os.ReadFile(tmp)
	require.NoError(t, readErr)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	assert.Equal(t, map[string]any{"name": "kv", "location": "eastus"}, got)
}

func TestExecute_Fields_AppliesToCSVFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[{"name":"a","location":"eastus","id":"/x/a"}]}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.csv")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.OutputFormat = "csv"
	cfg.Fields = []string{"name", "location"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/list")
	require.NoError(t, err)

	out, readErr := os.ReadFile(tmp)
	require.NoError(t, readErr)
	got := string(out)
	assert.Contains(t, got, "name")
	assert.Contains(t, got, "location")
	assert.NotContains(t, got, "/x/a") // id column dropped
}

func TestExecute_Fields_NonJSONLeftUnchanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("plain text body"))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.txt")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.Fields = []string{"name"}

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/plain")
	require.NoError(t, err)

	out, readErr := os.ReadFile(tmp)
	require.NoError(t, readErr)
	assert.Contains(t, string(out), "plain text body")
}

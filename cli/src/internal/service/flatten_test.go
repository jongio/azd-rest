package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlattenJSONBody_NestedAndArrays(t *testing.T) {
	got, err := flattenJSONBody([]byte(`{"a":{"b":1},"c":[10,20]}`))
	require.NoError(t, err)
	assert.Equal(t, `{"a.b":1,"c[0]":10,"c[1]":20}`, string(got))
}

func TestFlattenJSONBody_ScalarsKeepType(t *testing.T) {
	got, err := flattenJSONBody([]byte(`{"s":"x","n":3,"b":true,"z":null}`))
	require.NoError(t, err)
	assert.Equal(t, `{"b":true,"n":3,"s":"x","z":null}`, string(got))
}

func TestFlattenJSONBody_LargeIntKeepsPrecision(t *testing.T) {
	got, err := flattenJSONBody([]byte(`{"id":12345678901234567890}`))
	require.NoError(t, err)
	assert.Equal(t, `{"id":12345678901234567890}`, string(got))
}

func TestFlattenJSONBody_EmptyContainersKept(t *testing.T) {
	got, err := flattenJSONBody([]byte(`{"a":{},"b":[]}`))
	require.NoError(t, err)
	assert.Equal(t, `{"a":{},"b":[]}`, string(got))
}

func TestFlattenJSONBody_TopLevelArray(t *testing.T) {
	got, err := flattenJSONBody([]byte(`[{"name":"x"}]`))
	require.NoError(t, err)
	assert.Equal(t, `{"[0].name":"x"}`, string(got))
}

func TestFlattenJSONBody_TopLevelScalarUnchanged(t *testing.T) {
	got, err := flattenJSONBody([]byte(`42`))
	require.NoError(t, err)
	assert.Equal(t, `42`, string(got))
}

func TestFlattenJSONBody_InvalidJSON(t *testing.T) {
	_, err := flattenJSONBody([]byte(`not json`))
	require.Error(t, err)
}

func TestExecute_Flatten_JSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"properties":{"provisioningState":"Succeeded"},"tags":["a","b"]}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.json")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.Flatten = true

	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/r"))

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	body := string(out)

	assert.Contains(t, body, "properties.provisioningState")
	assert.Contains(t, body, "Succeeded")
	assert.Contains(t, body, "tags[0]")
	assert.NotContains(t, body, "\"properties\": {")
}

func TestExecute_Flatten_ComposesWithQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[{"name":"x"},{"name":"y"}]}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.json")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.Flatten = true
	cfg.Query = "value"

	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/r"))

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	body := string(out)

	assert.Contains(t, body, "[0].name")
	assert.Contains(t, body, "[1].name")
	assert.Contains(t, body, "x")
	assert.Contains(t, body, "y")
}

func TestExecute_Flatten_NonJSONLeftUnchanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json at all"))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.txt")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.Flatten = true

	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/r"))

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	assert.Contains(t, string(out), "not json at all")
}

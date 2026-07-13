package service

import (
	"context"
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

func TestCountRecords(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "array", body: `[1,2,3]`, want: 3},
		{name: "empty array", body: `[]`, want: 0},
		{name: "arm value wrapper", body: `{"value":[{"a":1},{"a":2}],"nextLink":"x"}`, want: 2},
		{name: "single object", body: `{"name":"kv"}`, want: 1},
		{name: "null", body: `null`, want: 0},
		{name: "bare number", body: `42`, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := countRecords([]byte(tt.body))
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountRecords_NonJSONError(t *testing.T) {
	_, err := countRecords([]byte("not json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON")
}

func TestExecute_Count_Array(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[{"name":"a"},{"name":"b"},{"name":"c"}]}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.txt")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.OutputFormat = "yaml"
	cfg.Count = true

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/list")
	require.NoError(t, err)

	out, readErr := os.ReadFile(tmp)
	require.NoError(t, readErr)
	got := strings.TrimSpace(string(out))
	assert.Equal(t, "3", got)
	assert.NotContains(t, string(out), "name") // body suppressed
}

func TestExecute_Count_RunsAfterQuery(t *testing.T) {
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
	cfg.Count = true

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/list")
	require.NoError(t, err)

	out, readErr := os.ReadFile(tmp)
	require.NoError(t, readErr)
	assert.Equal(t, "2", strings.TrimSpace(string(out)))
}

func TestExecute_Count_NonJSONReportsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("plain text"))
	}))
	defer srv.Close()

	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = filepath.Join(t.TempDir(), "out.txt")
	cfg.Count = true

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/plain")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON")
}

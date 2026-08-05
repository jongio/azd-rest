package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompactJSONBody(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		want   string
		wantOK bool
	}{
		{"object", "{\n  \"name\": \"web\",\n  \"count\": 2\n}", `{"name":"web","count":2}`, true},
		{"array", "[1, 2, 3]", `[1,2,3]`, true},
		{"nested", `{ "a": { "b": [1, 2] } }`, `{"a":{"b":[1,2]}}`, true},
		{"scalar string", `"hello"`, `"hello"`, true},
		{"not json", `plain text`, "", false},
		{"empty", ``, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := compactJSONBody([]byte(tc.body))
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExecute_Compact_Object(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\n  \"name\": \"web\",\n  \"count\": 2\n}"))
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.Compact = true

	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))

	got, err := os.ReadFile(cfg.OutputFile)
	require.NoError(t, err)
	assert.Equal(t, "{\"name\":\"web\",\"count\":2}\n", string(got))
}

func TestExecute_Compact_Array(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[1, 2, 3]"))
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.Compact = true

	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))

	got, err := os.ReadFile(cfg.OutputFile)
	require.NoError(t, err)
	assert.Equal(t, "[1,2,3]\n", string(got))
}

// TestExecute_Compact_WithQuery verifies --compact composes with --query.
func TestExecute_Compact_WithQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"props":{"x":1,"y":2}}`))
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.Compact = true
	cfg.Query = "props"

	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))

	got, err := os.ReadFile(cfg.OutputFile)
	require.NoError(t, err)
	assert.Equal(t, "{\"x\":1,\"y\":2}\n", string(got))
}

// TestExecute_Compact_NonJSON verifies a non-JSON response is left unchanged.
func TestExecute_Compact_NonJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("plain text"))
	}))
	defer srv.Close()

	cfg := baseTestConfig(t)
	cfg.Compact = true

	require.NoError(t, newTestService().Execute(context.Background(), cfg, "GET", srv.URL))

	got, err := os.ReadFile(cfg.OutputFile)
	require.NoError(t, err)
	assert.Equal(t, "plain text", string(got))
}

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

func TestRedactSecretsJSONBody_MasksSensitiveKeys(t *testing.T) {
	body := []byte(`{"name":"kv","password":"p","connectionString":"c","account_key":"k","sasToken":"s","accessToken":"a"}`)
	got, err := redactSecretsJSONBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := string(got)
	for _, secret := range []string{`"p"`, `"c"`, `"k"`, `"s"`, `"a"`} {
		if strings.Contains(out, secret) {
			t.Fatalf("secret value %s was not masked: %s", secret, out)
		}
	}
	if !strings.Contains(out, `"name":"kv"`) {
		t.Fatalf("non-sensitive field was changed: %s", out)
	}
	if strings.Count(out, redactedPlaceholder) != 5 {
		t.Fatalf("expected 5 masked values, got: %s", out)
	}
}

func TestRedactSecretsJSONBody_PreservesNonSensitiveKeys(t *testing.T) {
	body := []byte(`{"id":"1","location":"eastus","partitionKey":"pk","tokenType":"Bearer","displayName":"n"}`)
	got, err := redactSecretsJSONBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"displayName":"n","id":"1","location":"eastus","partitionKey":"pk","tokenType":"Bearer"}` {
		t.Fatalf("non-sensitive keys should be untouched, got: %s", string(got))
	}
}

func TestRedactSecretsJSONBody_NestedAndArrays(t *testing.T) {
	body := []byte(`{"items":[{"name":"a","clientSecret":"x"},{"name":"b","clientSecret":"y"}],"nested":{"apiKey":"z"}}`)
	got, err := redactSecretsJSONBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := string(got)
	for _, secret := range []string{`"x"`, `"y"`, `"z"`} {
		if strings.Contains(out, secret) {
			t.Fatalf("nested secret %s not masked: %s", secret, out)
		}
	}
	if strings.Count(out, redactedPlaceholder) != 3 {
		t.Fatalf("expected 3 masked values, got: %s", out)
	}
}

func TestRedactSecretsJSONBody_MasksWholeSubtree(t *testing.T) {
	body := []byte(`{"credentials":{"user":"u","pass":"p"},"public":"ok"}`)
	got, err := redactSecretsJSONBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"credentials":"REDACTED","public":"ok"}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestRedactSecretsJSONBody_CaseInsensitive(t *testing.T) {
	body := []byte(`{"PASSWORD":"p","Connection-String":"c"}`)
	got, err := redactSecretsJSONBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := string(got)
	if strings.Contains(out, `"p"`) || strings.Contains(out, `"c"`) {
		t.Fatalf("expected case-insensitive masking, got: %s", out)
	}
}

func TestRedactSecretsJSONBody_PreservesNumbers(t *testing.T) {
	body := []byte(`{"port":8443,"password":"x"}`)
	got, err := redactSecretsJSONBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), `"port":8443`) {
		t.Fatalf("expected port to stay numeric, got %q", string(got))
	}
	if !strings.Contains(string(got), `"password":"REDACTED"`) {
		t.Fatalf("expected password redacted, got %q", string(got))
	}
}

func TestRedactSecretsJSONBody_InvalidJSON(t *testing.T) {
	if _, err := redactSecretsJSONBody([]byte("not json")); err == nil {
		t.Fatal("expected error for non-JSON body")
	}
}

func TestExecute_RedactSecrets_MasksSensitiveField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"kv","connectionString":"s3cr3t"}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.json")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFile = tmp
	cfg.RedactSecrets = true

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/secret")
	require.NoError(t, err)

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	body := string(out)

	assert.Contains(t, body, "REDACTED")
	assert.NotContains(t, body, "s3cr3t")
	assert.Contains(t, body, "kv")
}

func TestExecute_RedactSecrets_RawLeftUnchanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"password":"s3cr3t"}`))
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "out.txt")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFormat = "raw"
	cfg.OutputFile = tmp
	cfg.RedactSecrets = true

	err := newTestService().Execute(context.Background(), cfg, "GET", srv.URL+"/secret")
	require.NoError(t, err)

	out, err := os.ReadFile(tmp)
	require.NoError(t, err)
	body := string(out)

	assert.Contains(t, body, "s3cr3t")
	assert.NotContains(t, body, "REDACTED")
}

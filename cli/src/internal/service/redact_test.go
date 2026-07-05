package service

import (
	"strings"
	"testing"
)

func TestRedactJSONBodyObjectPath(t *testing.T) {
	body := []byte(`{"name":"kv","value":"s3cr3t","nested":{"token":"abc"}}`)
	got, err := redactJSONBody(body, []string{"value", "nested.token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"name":"kv","nested":{"token":"REDACTED"},"value":"REDACTED"}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestRedactJSONBodyArrayWildcard(t *testing.T) {
	body := []byte(`{"value":[{"properties":{"secret":"a"}},{"properties":{"secret":"b"}}]}`)
	got, err := redactJSONBody(body, []string{"value.*.properties.secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"value":[{"properties":{"secret":"REDACTED"}},{"properties":{"secret":"REDACTED"}}]}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestRedactJSONBodyWildcardReplacesElements(t *testing.T) {
	body := []byte(`{"keys":["one","two"]}`)
	got, err := redactJSONBody(body, []string{"keys.*"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"keys":["REDACTED","REDACTED"]}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestRedactJSONBodyMissIsNoOp(t *testing.T) {
	body := []byte(`{"name":"kv"}`)
	got, err := redactJSONBody(body, []string{"value", "missing.deep.path", "name.*"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"name":"kv"}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestRedactJSONBodyPreservesNumbers(t *testing.T) {
	body := []byte(`{"port":8443,"secret":"x"}`)
	got, err := redactJSONBody(body, []string{"secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), `"port":8443`) {
		t.Fatalf("expected port to stay numeric, got %q", string(got))
	}
	if !strings.Contains(string(got), `"secret":"REDACTED"`) {
		t.Fatalf("expected secret redacted, got %q", string(got))
	}
}

func TestRedactJSONBodyInvalidJSON(t *testing.T) {
	if _, err := redactJSONBody([]byte("not json"), []string{"value"}); err == nil {
		t.Fatal("expected error for non-JSON body")
	}
}

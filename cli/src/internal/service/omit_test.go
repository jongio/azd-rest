package service

import "testing"

func TestOmitJSONBodyObjectPath(t *testing.T) {
	body := []byte(`{"name":"kv","value":"s3cr3t","nested":{"token":"abc","keep":"yes"}}`)
	got, err := omitJSONBody(body, []string{"value", "nested.token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"name":"kv","nested":{"keep":"yes"}}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestOmitJSONBodyArrayWildcard(t *testing.T) {
	body := []byte(`{"value":[{"properties":{"secret":"a","name":"x"}},{"properties":{"secret":"b","name":"y"}}]}`)
	got, err := omitJSONBody(body, []string{"value.*.properties.secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"value":[{"properties":{"name":"x"}},{"properties":{"name":"y"}}]}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestOmitJSONBodyWildcardEmptiesArray(t *testing.T) {
	body := []byte(`{"keys":["one","two"],"name":"kv"}`)
	got, err := omitJSONBody(body, []string{"keys.*"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"keys":[],"name":"kv"}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestOmitJSONBodyMissIsNoOp(t *testing.T) {
	body := []byte(`{"name":"kv"}`)
	got, err := omitJSONBody(body, []string{"value", "missing.deep.path", "name.*"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"name":"kv"}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestOmitJSONBodyPreservesNumbers(t *testing.T) {
	body := []byte(`{"port":8443,"secret":"x"}`)
	got, err := omitJSONBody(body, []string{"secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"port":8443}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestOmitJSONBodyTopLevelKey(t *testing.T) {
	body := []byte(`{"a":1,"b":2}`)
	got, err := omitJSONBody(body, []string{"a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"b":2}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestOmitJSONBodyInvalidJSON(t *testing.T) {
	if _, err := omitJSONBody([]byte("not json"), []string{"value"}); err == nil {
		t.Fatal("expected error for non-JSON body")
	}
}

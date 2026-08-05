package service

import (
	"encoding/json"
	"testing"
)

func TestLimitJSONBody_TopLevelArray(t *testing.T) {
	out, changed, err := limitJSONBody([]byte(`[{"name":"one"},{"name":"two"},{"name":"three"}]`), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected body to change")
	}

	var got []map[string]string
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(got) != 2 || got[1]["name"] != "two" {
		t.Fatalf("unexpected limited output: %#v", got)
	}
}

func TestLimitJSONBody_ARMValueArray(t *testing.T) {
	out, changed, err := limitJSONBody([]byte(`{"value":[{"name":"one"},{"name":"two"}],"nextLink":"https://example.com"}`), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected body to change")
	}

	var got struct {
		Value    []map[string]string `json:"value"`
		NextLink string              `json:"nextLink"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(got.Value) != 1 || got.Value[0]["name"] != "one" || got.NextLink == "" {
		t.Fatalf("unexpected limited output: %#v", got)
	}
}

func TestLimitJSONBody_NoCollection(t *testing.T) {
	body := []byte(`{"name":"one"}`)
	out, changed, err := limitJSONBody(body, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("did not expect scalar object to change")
	}
	if string(out) != string(body) {
		t.Fatalf("unexpected output: %s", out)
	}
}

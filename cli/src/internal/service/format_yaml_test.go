package service

import (
	"strings"
	"testing"
)

func TestRenderYAMLObject(t *testing.T) {
	body := `{"name":"web","location":"eastus","enabled":true}`
	out, err := renderYAML([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"name: web", "location: eastus", "enabled: true"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderYAMLSortsKeys(t *testing.T) {
	body := `{"b":2,"a":1,"c":3}`
	out, err := renderYAML([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "a: 1\nb: 2\nc: 3\n"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestRenderYAMLPreservesLargeInteger(t *testing.T) {
	body := `{"id":9007199254740993}`
	out, err := renderYAML([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "id: 9007199254740993") {
		t.Errorf("large integer not preserved:\n%s", out)
	}
}

func TestRenderYAMLArray(t *testing.T) {
	body := `[{"name":"a"},{"name":"b"}]`
	out, err := renderYAML([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "- name: a") || !strings.Contains(out, "- name: b") {
		t.Errorf("array not rendered as YAML list:\n%s", out)
	}
}

func TestRenderYAMLEmptyBody(t *testing.T) {
	out, err := renderYAML([]byte("   "))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestRenderYAMLInvalidJSON(t *testing.T) {
	if _, err := renderYAML([]byte("not json")); err == nil {
		t.Fatal("expected an error for non-JSON body")
	}
}

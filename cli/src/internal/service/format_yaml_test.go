package service

import (
	"strings"
	"testing"
)

func TestRenderYAMLObjectArray(t *testing.T) {
	body := []byte(`[{"name":"alpha","count":2},{"name":"beta","count":5}]`)
	out, err := renderYAML(body)
	if err != nil {
		t.Fatalf("renderYAML returned error: %v", err)
	}
	want := "- count: 2\n  name: alpha\n- count: 5\n  name: beta\n"
	if out != want {
		t.Fatalf("unexpected yaml:\n%q\nwant:\n%q", out, want)
	}
}

func TestRenderYAMLValueWrapper(t *testing.T) {
	body := []byte(`{"value":[{"id":"a"},{"id":"b"}],"nextLink":"https://example"}`)
	out, err := renderYAML(body)
	if err != nil {
		t.Fatalf("renderYAML returned error: %v", err)
	}
	want := "- id: a\n- id: b\n"
	if out != want {
		t.Fatalf("unexpected yaml:\n%q\nwant:\n%q", out, want)
	}
}

func TestRenderYAMLSingleObject(t *testing.T) {
	body := []byte(`{"name":"alpha","port":443,"nested":{"enabled":true}}`)
	out, err := renderYAML(body)
	if err != nil {
		t.Fatalf("renderYAML returned error: %v", err)
	}
	want := "name: alpha\nnested:\n  enabled: true\nport: 443\n"
	if out != want {
		t.Fatalf("unexpected yaml:\n%q\nwant:\n%q", out, want)
	}
}

func TestRenderYAMLNumbersNotQuoted(t *testing.T) {
	body := []byte(`{"port":8080,"ratio":1.5,"big":9007199254740993}`)
	out, err := renderYAML(body)
	if err != nil {
		t.Fatalf("renderYAML returned error: %v", err)
	}
	for _, want := range []string{"port: 8080", "ratio: 1.5", "big: 9007199254740993"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in yaml output, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, `"8080"`) {
		t.Fatalf("numbers should not be quoted, got:\n%s", out)
	}
}

func TestRenderYAMLTwoSpaceIndent(t *testing.T) {
	body := []byte(`{"outer":{"inner":"leaf"}}`)
	out, err := renderYAML(body)
	if err != nil {
		t.Fatalf("renderYAML returned error: %v", err)
	}
	if !strings.Contains(out, "\n  inner: leaf") {
		t.Fatalf("expected two-space indentation, got:\n%s", out)
	}
}

func TestRenderYAMLScalarArray(t *testing.T) {
	body := []byte(`["east","west"]`)
	out, err := renderYAML(body)
	if err != nil {
		t.Fatalf("renderYAML returned error: %v", err)
	}
	want := "- east\n- west\n"
	if out != want {
		t.Fatalf("unexpected yaml:\n%q\nwant:\n%q", out, want)
	}
}

func TestRenderYAMLInvalidJSON(t *testing.T) {
	if _, err := renderYAML([]byte("not json")); err == nil {
		t.Fatal("expected error for non-JSON body, got nil")
	}
}

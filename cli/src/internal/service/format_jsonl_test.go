package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func jsonlLines(s string) []string {
	trimmed := strings.TrimRight(s, "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func TestRenderJSONLTopLevelArray(t *testing.T) {
	body := `[{"name":"a"},{"name":"b"},{"name":"c"}]`
	out, err := renderJSONL([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := jsonlLines(out)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), out)
	}
	for i, want := range []string{`{"name":"a"}`, `{"name":"b"}`, `{"name":"c"}`} {
		if lines[i] != want {
			t.Errorf("line %d = %q, want %q", i, lines[i], want)
		}
	}
}

func TestRenderJSONLWrappedInValue(t *testing.T) {
	body := `{"value":[{"id":1},{"id":2}],"nextLink":"..."}`
	out, err := renderJSONL([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := jsonlLines(out)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines from value[], got %d:\n%s", len(lines), out)
	}
	if lines[0] != `{"id":1}` {
		t.Errorf("line 0 = %q", lines[0])
	}
}

func TestRenderJSONLResourceGraphData(t *testing.T) {
	body := `{"totalRecords":2,"data":[{"name":"x"},{"name":"y"}]}`
	out, err := renderJSONL([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jsonlLines(out)) != 2 {
		t.Fatalf("expected 2 lines from data[]:\n%s", out)
	}
}

func TestRenderJSONLSingleObject(t *testing.T) {
	body := `{"name":"solo","enabled":true}`
	out, err := renderJSONL([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := jsonlLines(out)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d:\n%s", len(lines), out)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &obj); err != nil {
		t.Fatalf("line is not valid JSON: %v", err)
	}
}

func TestRenderJSONLPreservesLargeIntegers(t *testing.T) {
	// A float64 round-trip would turn this into 1e+15; UseNumber must keep it exact.
	body := `[{"id":1000000000000000}]`
	out, err := renderJSONL([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "1000000000000000") {
		t.Errorf("large integer not preserved:\n%s", out)
	}
}

func TestRenderJSONLDoesNotEscapeHTML(t *testing.T) {
	body := `[{"url":"https://example.com/a?x=1&y=2"}]`
	out, err := renderJSONL([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, `\u0026`) {
		t.Errorf("ampersand should not be HTML-escaped:\n%s", out)
	}
	if !strings.Contains(out, "x=1&y=2") {
		t.Errorf("expected raw ampersand in output:\n%s", out)
	}
}

func TestRenderJSONLEmptyArray(t *testing.T) {
	out, err := renderJSONL([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output for empty array, got: %q", out)
	}
}

func TestRenderJSONLInvalidJSON(t *testing.T) {
	if _, err := renderJSONL([]byte(`not json`)); err == nil {
		t.Fatalf("expected error for non-JSON body")
	}
}

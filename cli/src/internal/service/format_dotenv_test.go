package service

import (
	"strings"
	"testing"
)

// parseDotenv splits dotenv output into a key to value map for assertions. It
// keeps the raw right-hand side (quotes and escapes intact) so tests can assert
// on the exact rendered form.
func parseDotenv(t *testing.T, s string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		if line == "" {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			t.Fatalf("line has no '=': %q", line)
		}
		out[line[:eq]] = line[eq+1:]
	}
	return out
}

func TestRenderDotenvFlatObject(t *testing.T) {
	body := `{"name":"web","port":8080,"enabled":true,"note":null}`
	out, err := renderDotenv([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := parseDotenv(t, out)
	if got["NAME"] != "web" {
		t.Errorf("NAME: got %q", got["NAME"])
	}
	if got["PORT"] != "8080" {
		t.Errorf("PORT: got %q", got["PORT"])
	}
	if got["ENABLED"] != "true" {
		t.Errorf("ENABLED: got %q", got["ENABLED"])
	}
	if _, ok := got["NOTE"]; !ok || got["NOTE"] != "" {
		t.Errorf("NOTE should be present and empty, got %q (present %v)", got["NOTE"], ok)
	}
}

func TestRenderDotenvNestedToDottedKeys(t *testing.T) {
	body := `{"properties":{"state":"Succeeded"}}`
	out, err := renderDotenv([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := parseDotenv(t, out)
	if got["PROPERTIES_STATE"] != "Succeeded" {
		t.Errorf("nested key not mapped: %v", got)
	}
}

func TestRenderDotenvArrayIndexKeys(t *testing.T) {
	body := `{"value":[{"name":"a"},{"name":"b"}]}`
	out, err := renderDotenv([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := parseDotenv(t, out)
	if got["VALUE_0_NAME"] != "a" || got["VALUE_1_NAME"] != "b" {
		t.Errorf("array index keys not mapped: %v", got)
	}
}

func TestRenderDotenvTopLevelArray(t *testing.T) {
	body := `[{"name":"a"}]`
	out, err := renderDotenv([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := parseDotenv(t, out)
	// A leading digit is prefixed with an underscore to stay a valid identifier.
	if got["_0_NAME"] != "a" {
		t.Errorf("top-level array key not mapped: %v", got)
	}
}

func TestRenderDotenvQuotesUnsafeValues(t *testing.T) {
	body := `{"msg":"hello world","path":"a=b#c","tab":"x\ty"}`
	out, err := renderDotenv([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := parseDotenv(t, out)
	if got["MSG"] != `"hello world"` {
		t.Errorf("space value should be quoted, got %q", got["MSG"])
	}
	if got["PATH"] != `"a=b#c"` {
		t.Errorf("special chars should be quoted, got %q", got["PATH"])
	}
	if got["TAB"] != `"x\ty"` {
		t.Errorf("tab should be escaped, got %q", got["TAB"])
	}
}

func TestRenderDotenvEscapesQuotesAndBackslash(t *testing.T) {
	body := `{"note":"say \"hi\" \\ done"}`
	out, err := renderDotenv([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := parseDotenv(t, out)
	if got["NOTE"] != `"say \"hi\" \\ done"` {
		t.Errorf("quotes/backslash not escaped, got %q", got["NOTE"])
	}
}

func TestRenderDotenvSafeStringUnquoted(t *testing.T) {
	body := `{"id":"resource-group-01"}`
	out, err := renderDotenv([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := parseDotenv(t, out)
	if got["ID"] != "resource-group-01" {
		t.Errorf("safe string should be bare, got %q", got["ID"])
	}
}

func TestRenderDotenvSortedDeterministic(t *testing.T) {
	body := `{"b":"2","a":"1","c":"3"}`
	out, err := renderDotenv([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "A=1\nB=2\nC=3\n" {
		t.Errorf("output not sorted deterministically: %q", out)
	}
}

func TestRenderDotenvBareScalarError(t *testing.T) {
	if _, err := renderDotenv([]byte(`"just a string"`)); err == nil {
		t.Fatalf("expected error for bare scalar body")
	}
}

func TestRenderDotenvInvalidJSON(t *testing.T) {
	if _, err := renderDotenv([]byte(`not json`)); err == nil {
		t.Fatalf("expected error for non-JSON body")
	}
}

package service

import (
	"regexp"
	"strings"
	"testing"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func TestColorizeJSON_PreservesStructure(t *testing.T) {
	in := "{\n  \"name\": \"value\",\n  \"count\": 42,\n  \"ok\": true,\n  \"empty\": null,\n  \"neg\": -1.5\n}"
	out := colorizeJSON(in)
	if stripANSI(out) != in {
		t.Fatalf("stripping ANSI did not reproduce input.\n got: %q\nwant: %q", stripANSI(out), in)
	}
	for _, code := range []string{colorKey, colorString, colorNumber, colorLit} {
		if !strings.Contains(out, code) {
			t.Fatalf("expected color code %q in output: %q", code, out)
		}
	}
}

func TestColorizeJSON_KeyVsStringValue(t *testing.T) {
	in := `{"k": "v"}`
	out := colorizeJSON(in)
	if !strings.Contains(out, colorKey+`"k"`+colorReset) {
		t.Errorf("key not colored as key: %q", out)
	}
	if !strings.Contains(out, colorString+`"v"`+colorReset) {
		t.Errorf("string value not colored as string: %q", out)
	}
}

func TestValidateColorMode(t *testing.T) {
	for _, m := range []string{"", "auto", "always", "never"} {
		if err := validateColorMode(m); err != nil {
			t.Errorf("validateColorMode(%q) unexpected error: %v", m, err)
		}
	}
	if err := validateColorMode("rainbow"); err == nil {
		t.Errorf("expected error for invalid mode")
	}
}

func TestShouldColorize(t *testing.T) {
	jsonResp := &client.Response{Body: []byte(`{"a":1}`)}
	nonJSON := &client.Response{Body: []byte(`not json`)}

	tests := []struct {
		name string
		cfg  config.Config
		resp *client.Response
		want bool
	}{
		{"never", config.Config{Color: "never"}, jsonResp, false},
		{"always json stdout", config.Config{Color: "always"}, jsonResp, true},
		{"always but output-file", config.Config{Color: "always", OutputFile: "out.json"}, jsonResp, false},
		{"always but raw format", config.Config{Color: "always", OutputFormat: "raw"}, jsonResp, false},
		{"always but non-json", config.Config{Color: "always"}, nonJSON, false},
		{"auto non-tty", config.Config{Color: "auto"}, jsonResp, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldColorize(tt.cfg, tt.resp); got != tt.want {
				t.Errorf("shouldColorize = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldColorize_NoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	jsonResp := &client.Response{Body: []byte(`{"a":1}`)}
	if shouldColorize(config.Config{Color: "auto"}, jsonResp) {
		t.Errorf("auto mode should not colorize when NO_COLOR is set")
	}
}

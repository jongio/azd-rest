package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// newConfigTestFlags builds a flag set that mimics the persistent flags the
// config command reports, so tests exercise collection without the full root
// command.
func newConfigTestFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("format", "auto", "")
	fs.Int("retry", 3, "")
	fs.StringArray("allow-host", []string{}, "")
	return fs
}

func TestCollectConfigEntries_DefaultsOnly(t *testing.T) {
	fs := newConfigTestFlags()
	names := []string{"format", "retry", "allow-host"}
	lookup := func(string) (string, bool) { return "", false }

	entries := collectConfigEntries(fs, names, lookup)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Entries are sorted by flag name: allow-host, format, retry.
	if entries[0].Flag != "allow-host" || entries[1].Flag != "format" || entries[2].Flag != "retry" {
		t.Fatalf("entries not sorted by flag name: %#v", entries)
	}

	format := entries[1]
	if format.Default != "auto" || format.Value != "auto" {
		t.Fatalf("unexpected format default/value: %#v", format)
	}
	if format.EnvVar != "AZD_REST_FORMAT" {
		t.Fatalf("unexpected env var for format: %q", format.EnvVar)
	}
	if format.Source != sourceDefault {
		t.Fatalf("expected default source, got %q", format.Source)
	}
}

func TestCollectConfigEntries_EnvVarNames(t *testing.T) {
	fs := newConfigTestFlags()
	lookup := func(string) (string, bool) { return "", false }

	entries := collectConfigEntries(fs, []string{"allow-host"}, lookup)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// allow-host maps to the comma-separated AZD_REST_ALLOWED_HOSTS variable,
	// not the AZD_REST_ALLOW_HOST default the naming convention would produce.
	if entries[0].EnvVar != "AZD_REST_ALLOWED_HOSTS" {
		t.Fatalf("expected AZD_REST_ALLOWED_HOSTS, got %q", entries[0].EnvVar)
	}
}

func TestCollectConfigEntries_EnvironmentSource(t *testing.T) {
	fs := newConfigTestFlags()
	// Simulate the value the persistent pre-run would have already applied.
	if err := fs.Set("retry", "5"); err != nil {
		t.Fatalf("set retry: %v", err)
	}
	lookup := func(name string) (string, bool) {
		if name == "AZD_REST_RETRY" {
			return "5", true
		}
		return "", false
	}

	entries := collectConfigEntries(fs, []string{"retry"}, lookup)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Source != sourceEnvironment {
		t.Fatalf("expected environment source, got %q", e.Source)
	}
	if e.Value != "5" {
		t.Fatalf("expected effective value 5, got %q", e.Value)
	}
	if e.Default != "3" {
		t.Fatalf("expected default 3, got %q", e.Default)
	}
}

func TestCollectConfigEntries_EmptyEnvIsDefault(t *testing.T) {
	fs := newConfigTestFlags()
	// An env var that is present but empty must not be treated as a source, so
	// it matches how applyEnvDefaults ignores empty values.
	lookup := func(name string) (string, bool) {
		if name == "AZD_REST_FORMAT" {
			return "", true
		}
		return "", false
	}

	entries := collectConfigEntries(fs, []string{"format"}, lookup)
	if entries[0].Source != sourceDefault {
		t.Fatalf("expected default source for empty env var, got %q", entries[0].Source)
	}
}

func TestCollectConfigEntries_SkipsUnknownFlag(t *testing.T) {
	fs := newConfigTestFlags()
	lookup := func(string) (string, bool) { return "", false }

	entries := collectConfigEntries(fs, []string{"format", "does-not-exist"}, lookup)
	if len(entries) != 1 {
		t.Fatalf("expected unknown flag to be skipped, got %d entries", len(entries))
	}
	if entries[0].Flag != "format" {
		t.Fatalf("unexpected entry: %#v", entries[0])
	}
}

func TestRunConfig_JSON(t *testing.T) {
	fs := newConfigTestFlags()
	lookup := func(string) (string, bool) { return "", false }
	var buf bytes.Buffer

	if err := runConfig(fs, []string{"format", "retry"}, lookup, formatJSON, &buf); err != nil {
		t.Fatalf("runConfig: %v", err)
	}

	var entries []configEntry
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestRunConfig_Text(t *testing.T) {
	fs := newConfigTestFlags()
	lookup := func(string) (string, bool) { return "", false }
	var buf bytes.Buffer

	if err := runConfig(fs, []string{"format", "retry"}, lookup, "auto", &buf); err != nil {
		t.Fatalf("runConfig: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "FLAG") || !strings.Contains(out, "ENV VAR") {
		t.Fatalf("text output missing header: %q", out)
	}
	if !strings.Contains(out, "AZD_REST_FORMAT") {
		t.Fatalf("text output missing env var mapping: %q", out)
	}
}

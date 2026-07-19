package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// jsonNull is the JSON literal for a null value.
const jsonNull = "null"

// renderDotenv renders a JSON response as dotenv-style KEY=value lines for a
// shell eval, a CI job's environment, or a docker --env-file.
//
// The body is flattened with flattenJSONBody so nested fields become dotted
// paths, then each path is mapped to an uppercase, underscore-separated key
// (properties.state becomes PROPERTIES_STATE, value[0].name becomes
// VALUE_0_NAME). String values are emitted bare when they
// are already shell-safe and double-quoted with escaping otherwise. Numbers and
// booleans render as their literal text, and null renders as an empty value.
// Keys are sorted for deterministic output. A response that is not a JSON object
// or array has no field names to emit, so it returns an error.
func renderDotenv(body []byte) (string, error) {
	flat, err := flattenJSONBody(body)
	if err != nil {
		return "", fmt.Errorf("dotenv format requires a JSON response: %w", err)
	}

	fields := map[string]json.RawMessage{}
	dec := json.NewDecoder(bytes.NewReader(flat))
	dec.UseNumber()
	if err := dec.Decode(&fields); err != nil {
		return "", fmt.Errorf("dotenv format requires a JSON object or array response: %w", err)
	}

	paths := make([]string, 0, len(fields))
	for p := range fields {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var b strings.Builder
	for _, p := range paths {
		value, err := dotenvValue(fields[p])
		if err != nil {
			return "", err
		}
		b.WriteString(dotenvKey(p))
		b.WriteByte('=')
		b.WriteString(value)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// dotenvKey converts a flattened dotted path into a shell-safe environment
// variable name: every run of characters outside [A-Za-z0-9] collapses to a
// single underscore, surrounding underscores are trimmed, each letter is
// converted to upper case, and a leading digit is prefixed with an underscore
// so the name is a valid identifier.
func dotenvKey(path string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range path {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(toUpperASCII(r))
			prevUnderscore = false
		case !prevUnderscore:
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	key := strings.Trim(b.String(), "_")
	if key == "" {
		return "_"
	}
	if key[0] >= '0' && key[0] <= '9' {
		return "_" + key
	}
	return key
}

// toUpperASCII maps an ASCII letter to its upper case form and leaves every other rune unchanged.
func toUpperASCII(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}

// dotenvValue renders a single flattened leaf value as a dotenv right-hand side.
// null and an empty raw value become an empty string, JSON strings are unquoted
// then re-quoted only when needed, empty objects and arrays are emitted as
// compact JSON, and numbers and booleans pass through as safe literals.
func dotenvValue(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == jsonNull {
		return "", nil
	}
	switch trimmed[0] {
	case '"':
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return "", fmt.Errorf("failed to decode dotenv value: %w", err)
		}
		return dotenvQuote(s), nil
	case '{', '[':
		return dotenvQuote(string(trimmed)), nil
	default:
		return string(trimmed), nil
	}
}

// dotenvQuote returns s unquoted when it is a non-empty run of shell-safe
// characters, and otherwise wraps it in double quotes with backslash, quote, and
// control characters escaped so the line round-trips through common dotenv
// parsers.
func dotenvQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\r\n\"'\\#=$`") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

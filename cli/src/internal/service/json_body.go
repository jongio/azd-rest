package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// applicationJSON is the media type set for --json-field bodies.
const applicationJSON = "application/json"

// buildJSONBody assembles a JSON request body from repeatable --json-field and
// --json-field-raw flags.
//
// String fields use key=value and are stored as JSON strings. Raw fields use
// key:=json and are parsed as JSON so numbers, booleans, arrays, objects, and
// null keep their type. Dotted keys such as sku.name build nested objects, and
// repeated prefixes merge into the same parent object. String fields are
// applied first, then raw fields.
func buildJSONBody(stringFields, rawFields []string) ([]byte, error) {
	root := map[string]any{}

	for _, field := range stringFields {
		key, value, ok := strings.Cut(field, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --json-field format: %s (expected key=value)", field)
		}
		if err := setNestedValue(root, key, value); err != nil {
			return nil, err
		}
	}

	for _, field := range rawFields {
		key, raw, ok := strings.Cut(field, ":=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --json-field-raw format: %s (expected key:=json)", field)
		}
		dec := json.NewDecoder(strings.NewReader(raw))
		dec.UseNumber()
		var parsed any
		if err := dec.Decode(&parsed); err != nil {
			return nil, fmt.Errorf("invalid JSON for --json-field-raw %q: %w", key, err)
		}
		if dec.More() {
			return nil, fmt.Errorf("invalid JSON for --json-field-raw %q: unexpected trailing data", key)
		}
		if err := setNestedValue(root, key, parsed); err != nil {
			return nil, err
		}
	}

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("failed to encode json body: %w", err)
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// setNestedValue assigns value at a dotted key path within obj, creating
// intermediate objects as needed. It fails when a path segment is empty or when
// an existing non-object value blocks the path.
func setNestedValue(obj map[string]any, key string, value any) error {
	segments := strings.Split(key, ".")
	for _, seg := range segments {
		if seg == "" {
			return fmt.Errorf("invalid field key %q: empty path segment", key)
		}
	}

	current := obj
	for i, seg := range segments {
		if i == len(segments)-1 {
			current[seg] = value
			return nil
		}
		child, ok := current[seg]
		if !ok {
			next := map[string]any{}
			current[seg] = next
			current = next
			continue
		}
		nextMap, ok := child.(map[string]any)
		if !ok {
			return fmt.Errorf("cannot set %q: %q is already set to a non-object value", key, seg)
		}
		current = nextMap
	}
	return nil
}

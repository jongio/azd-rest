package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

// applicationJSON is the media type set for --json-field bodies.
const applicationJSON = "application/json"

// encodeJSONFields turns repeatable key=value flags into an application/json
// object body. A plain key=value pair sets a string value. A key:=value pair
// sets a raw JSON value, so count:=3, enabled:=true, and tags:=["a","b"]
// produce a number, boolean, and array. Repeated keys keep the last value.
func encodeJSONFields(fields []string) (string, error) {
	obj := make(map[string]any, len(fields))
	for _, field := range fields {
		key, value, raw, err := parseJSONField(field)
		if err != nil {
			return "", err
		}
		if raw {
			var parsed any
			if err := json.Unmarshal([]byte(value), &parsed); err != nil {
				return "", fmt.Errorf("invalid --json-field %q: value is not valid JSON: %w", field, err)
			}
			obj[key] = parsed
			continue
		}
		obj[key] = value
	}
	encoded, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to encode --json-field body: %w", err)
	}
	return string(encoded), nil
}

// parseJSONField splits one --json-field argument into its key and value and
// reports whether the value should be parsed as raw JSON. A key:=value pair is
// raw; a key=value pair is a string. The key must be non-empty.
func parseJSONField(field string) (key, value string, raw bool, err error) {
	eq := strings.IndexByte(field, '=')
	if eq < 1 {
		return "", "", false, fmt.Errorf("invalid --json-field format: %s (expected key=value or key:=value)", field)
	}
	if field[eq-1] == ':' {
		key = field[:eq-1]
		if key == "" {
			return "", "", false, fmt.Errorf("invalid --json-field format: %s (expected key=value or key:=value)", field)
		}
		return key, field[eq+1:], true, nil
	}
	return field[:eq], field[eq+1:], false, nil
}

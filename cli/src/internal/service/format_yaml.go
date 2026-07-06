package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// renderYAML renders a JSON response body as YAML with two-space indentation
// and stable (alphabetical) key order.
//
// A top-level array, or an object that wraps rows under a common key (value,
// data, results, or items), renders as a YAML sequence of those rows, matching
// the rows produced by the table and jsonl formats. Any other value, including
// a single resource object, renders directly as a YAML mapping or scalar.
// Numbers are converted from their JSON form so they are emitted as YAML
// numbers rather than quoted strings.
func renderYAML(body []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return "", fmt.Errorf("yaml format requires a JSON response: %w", err)
	}

	value := convertYAMLNumbers(unwrapListBody(parsed))

	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(value); err != nil {
		return "", fmt.Errorf("failed to encode response as yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", fmt.Errorf("failed to encode response as yaml: %w", err)
	}
	return b.String(), nil
}

// unwrapListBody returns the wrapped row array when parsed is a top-level array
// or a common wrapper object, so yaml renders the same rows as table and jsonl.
// All other values, including single objects, are returned unchanged.
func unwrapListBody(parsed any) any {
	switch v := parsed.(type) {
	case []any:
		return v
	case map[string]any:
		for _, key := range listWrapperKeys {
			if arr, ok := v[key].([]any); ok {
				return arr
			}
		}
		return v
	default:
		return v
	}
}

// convertYAMLNumbers walks a decoded JSON value and converts json.Number values
// into int64 or float64 so the YAML encoder emits them as numbers rather than
// quoted strings. Maps and slices are copied so the original value is untouched.
func convertYAMLNumbers(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, val := range v {
			out[key] = convertYAMLNumbers(val)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = convertYAMLNumbers(val)
		}
		return out
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i
		}
		if f, err := v.Float64(); err == nil {
			return f
		}
		return v.String()
	default:
		return v
	}
}

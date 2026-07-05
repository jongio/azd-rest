package service

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// renderYAML renders a JSON response body as YAML. Azure REST responses are
// JSON, and YAML is often easier to scan by eye for nested objects and to diff
// across runs. Map keys are sorted by the YAML encoder for stable output.
//
// Numbers are decoded with json.Number and converted back to integers or
// floats so large integer IDs keep their exact value instead of being rendered
// in floating point notation.
func renderYAML(body []byte) (string, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return "", nil
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return "", fmt.Errorf("yaml format requires a JSON response: %w", err)
	}

	out, err := yaml.Marshal(normalizeJSONNumbers(parsed))
	if err != nil {
		return "", fmt.Errorf("failed to encode response as YAML: %w", err)
	}
	return string(out), nil
}

// normalizeJSONNumbers walks a value decoded with json.Number and converts each
// json.Number to an int64 when it fits, otherwise a float64, so the YAML
// encoder emits plain numeric scalars instead of quoted strings.
func normalizeJSONNumbers(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for key, val := range t {
			t[key] = normalizeJSONNumbers(val)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = normalizeJSONNumbers(val)
		}
		return t
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	default:
		return v
	}
}

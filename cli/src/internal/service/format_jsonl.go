package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// renderJSONL renders a JSON response body as newline-delimited JSON (NDJSON):
// one compact JSON value per line. This is the format tools like jq -c, and
// log pipelines, consume most easily.
//
// It accepts a top-level JSON array (one line per element), an object that
// wraps rows under a common key (value, data, results, or items), or any other
// value (emitted as a single line). Numbers are preserved exactly so large
// integer IDs do not lose precision.
func renderJSONL(body []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return "", fmt.Errorf("jsonl format requires a JSON response: %w", err)
	}

	rows := extractJSONLRows(parsed)

	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return "", fmt.Errorf("failed to encode row as JSON: %w", err)
		}
	}
	return b.String(), nil
}

// extractJSONLRows normalizes a parsed JSON value into the rows to emit. Arrays
// and common wrapper objects are unwrapped so each element becomes its own
// line; any other value is emitted as a single line.
func extractJSONLRows(parsed any) []any {
	switch v := parsed.(type) {
	case []any:
		return v
	case map[string]any:
		for _, key := range listWrapperKeys {
			if arr, ok := v[key].([]any); ok {
				return arr
			}
		}
		return []any{v}
	default:
		return []any{parsed}
	}
}

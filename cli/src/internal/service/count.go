package service

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// countRecords returns the number of records in a JSON response body. A JSON
// array counts its elements, an ARM value[] wrapper counts the value array, a
// null counts as zero, and any other single value counts as one. A body that is
// not valid JSON returns an error so the caller can report it clearly.
func countRecords(body []byte) (int, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return 0, fmt.Errorf("--count needs a JSON response, but the body did not parse as JSON: %w", err)
	}

	switch node := parsed.(type) {
	case nil:
		return 0, nil
	case []any:
		return len(node), nil
	case map[string]any:
		if inner, ok := node["value"].([]any); ok {
			return len(inner), nil
		}
		return 1, nil
	default:
		return 1, nil
	}
}

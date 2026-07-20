package service

import (
	"bytes"
	"encoding/json"
)

// projectFields parses body as JSON and keeps only the listed top-level fields.
// It handles three shapes: a single object keeps the listed keys, an array of
// objects trims each element, and an ARM value[] wrapper trims each element of
// value[] while keeping the wrapper so paging links (nextLink) survive. A value
// that is not an object is left as-is.
//
// The second return value is false when the body is not valid JSON, so the
// caller can leave the response unchanged and print a note.
func projectFields(body []byte, fields []string) ([]byte, bool) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return body, false
	}

	keep := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		keep[f] = struct{}{}
	}

	projected := projectValue(parsed, keep)

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(projected); err != nil {
		return body, false
	}
	return bytes.TrimRight(b.Bytes(), "\n"), true
}

// projectValue applies the field filter to a decoded JSON value, dispatching on
// the three supported shapes.
func projectValue(v any, keep map[string]struct{}) any {
	switch node := v.(type) {
	case map[string]any:
		// ARM value[] wrapper: trim each element of value[] and keep the
		// wrapper (and its paging keys) so nextLink survives.
		if inner, ok := node["value"].([]any); ok {
			node["value"] = projectArray(inner, keep)
			return node
		}
		return projectObject(node, keep)
	case []any:
		return projectArray(node, keep)
	default:
		return v
	}
}

// projectArray trims each object element of an array, leaving non-object
// elements unchanged.
func projectArray(arr []any, keep map[string]struct{}) []any {
	for i, item := range arr {
		if obj, ok := item.(map[string]any); ok {
			arr[i] = projectObject(obj, keep)
		}
	}
	return arr
}

// projectObject returns a new object containing only the keys in keep that are
// present in the source object.
func projectObject(obj map[string]any, keep map[string]struct{}) map[string]any {
	out := make(map[string]any, len(keep))
	for k := range keep {
		if val, ok := obj[k]; ok {
			out[k] = val
		}
	}
	return out
}

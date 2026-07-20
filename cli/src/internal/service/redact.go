package service

import (
	"bytes"
	"encoding/json"
	"strings"
)

// redactedPlaceholder is the fixed value that replaces every redacted field.
const redactedPlaceholder = "REDACTED"

const (
	// formatRaw is the --format value for unparsed raw output.
	formatRaw = "raw"
	// formatXML is the --format value for XML output.
	formatXML = "xml"
)

// redactJSONBody parses body as JSON, replaces the value at each dotted path
// with a fixed placeholder, and returns the re-encoded JSON.
//
// Path segments are separated by dots. A "*" segment matches every element of
// an array. A path that matches nothing is ignored. An error is returned only
// when the body is not valid JSON, so callers can leave the body unchanged.
func redactJSONBody(body []byte, paths []string) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return nil, err
	}

	for _, path := range paths {
		segments := splitRedactPath(path)
		if len(segments) == 0 {
			continue
		}
		applyRedaction(parsed, segments)
	}

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(parsed); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// splitRedactPath breaks a dotted path into segments, ignoring an empty path.
func splitRedactPath(path string) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return strings.Split(path, ".")
}

// applyRedaction walks node following segments and replaces the matched value
// with the placeholder. A "*" segment iterates every element of an array. Any
// segment that does not match the current shape stops that branch without
// error, so a path that matches nothing is a no-op.
func applyRedaction(node any, segments []string) {
	if len(segments) == 0 {
		return
	}
	seg := segments[0]
	last := len(segments) == 1

	if seg == "*" {
		arr, ok := node.([]any)
		if !ok {
			return
		}
		for i := range arr {
			if last {
				arr[i] = redactedPlaceholder
			} else {
				applyRedaction(arr[i], segments[1:])
			}
		}
		return
	}

	obj, ok := node.(map[string]any)
	if !ok {
		return
	}
	child, exists := obj[seg]
	if !exists {
		return
	}
	if last {
		obj[seg] = redactedPlaceholder
		return
	}
	applyRedaction(child, segments[1:])
}

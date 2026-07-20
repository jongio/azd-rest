package service

import (
	"bytes"
	"encoding/json"
)

// omitJSONBody parses body as JSON, removes the field at each dotted path, and
// returns the re-encoded JSON. It is the structural complement to
// redactJSONBody: redaction masks a value in place, omission removes the key
// (or every element of a matched array) so the field no longer appears in the
// output.
//
// Path segments are separated by dots and follow the same rules as --redact. A
// "*" segment matches every element of an array. A path that matches nothing is
// ignored. An error is returned only when the body is not valid JSON, so callers
// can leave the body unchanged.
func omitJSONBody(body []byte, paths []string) ([]byte, error) {
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
		parsed = applyOmission(parsed, segments)
	}

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(parsed); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// applyOmission walks node following segments and removes the matched field or
// array elements, returning the possibly rebuilt node so the caller can reattach
// it. Removing an array element changes the slice length, so the new node is
// returned rather than mutated through the parent reference. A "*" segment
// iterates every element of an array. Any segment that does not match the
// current shape leaves that branch unchanged, so a path that matches nothing is
// a no-op.
func applyOmission(node any, segments []string) any {
	if len(segments) == 0 {
		return node
	}
	seg := segments[0]
	last := len(segments) == 1

	if seg == "*" {
		arr, ok := node.([]any)
		if !ok {
			return node
		}
		if last {
			// Omitting every element leaves an empty array.
			return []any{}
		}
		for i := range arr {
			arr[i] = applyOmission(arr[i], segments[1:])
		}
		return arr
	}

	obj, ok := node.(map[string]any)
	if !ok {
		return node
	}
	child, exists := obj[seg]
	if !exists {
		return node
	}
	if last {
		delete(obj, seg)
		return obj
	}
	obj[seg] = applyOmission(child, segments[1:])
	return obj
}

package service

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
)

// flattenJSONBody collapses a JSON document into a single-level object whose keys
// are dotted paths to each leaf value. Nested objects join with a dot
// (properties.provisioningState) and array elements use a bracket index
// (value[0].name). Leaf values keep their original JSON type, and numbers are
// decoded with json.Number so large resource identifiers keep full precision.
//
// A top-level value that is not an object or array (a bare scalar) is returned
// unchanged, since there is nothing to flatten.
func flattenJSONBody(body []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var root any
	if err := dec.Decode(&root); err != nil {
		return nil, err
	}

	switch root.(type) {
	case map[string]any, []any:
		// Flatten below.
	default:
		return body, nil
	}

	flat := make(map[string]json.RawMessage)
	if err := flattenValue("", root, flat); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(flat))
	for k := range flat {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(flat[k])
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// flattenValue walks a decoded JSON value and records each leaf under its dotted
// path in out. Empty objects and arrays are treated as leaves so they are not
// silently dropped.
func flattenValue(prefix string, v any, out map[string]json.RawMessage) error {
	switch node := v.(type) {
	case map[string]any:
		if len(node) == 0 {
			return setFlatLeaf(prefix, node, out)
		}
		for k, child := range node {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			if err := flattenValue(key, child, out); err != nil {
				return err
			}
		}
		return nil
	case []any:
		if len(node) == 0 {
			return setFlatLeaf(prefix, node, out)
		}
		for i, child := range node {
			key := prefix + "[" + strconv.Itoa(i) + "]"
			if err := flattenValue(key, child, out); err != nil {
				return err
			}
		}
		return nil
	default:
		return setFlatLeaf(prefix, node, out)
	}
}

// setFlatLeaf marshals a leaf value and stores it under prefix.
func setFlatLeaf(prefix string, v any, out map[string]json.RawMessage) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	out[prefix] = raw
	return nil
}

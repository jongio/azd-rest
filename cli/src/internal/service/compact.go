package service

import (
	"bytes"
	"encoding/json"
)

// compactJSONBody minifies a JSON document to a single line with no
// insignificant whitespace (#235). Key order and number formatting are
// preserved exactly, since json.Compact only strips whitespace. It returns
// ok=false when body is not valid JSON so the caller can leave the output
// unchanged.
func compactJSONBody(body []byte) (string, bool) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, body); err != nil {
		return "", false
	}
	return buf.String(), true
}

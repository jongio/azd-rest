package service

import (
	"bytes"
	"encoding/json"
	"strings"
)

// rawOutputUsageError signals that --raw-output was used without --query. It
// reports exit code 2 (the invalid-configuration code) through the ExitCoder
// contract so main can distinguish it from a request failure.
type rawOutputUsageError struct{ msg string }

func (e *rawOutputUsageError) Error() string { return e.msg }

// ExitCode returns 2 for invalid --raw-output usage.
func (e *rawOutputUsageError) ExitCode() int { return 2 }

// rawOutputText renders a --query result for --raw-output (#234), mirroring
// jq -r. A JSON string is returned unquoted with a trailing newline, and an
// array of strings is returned as one value per line. Any other shape (object,
// number, boolean, null, or a mixed array) returns ok=false so the caller keeps
// the normal JSON output instead of silently mangling it.
func rawOutputText(body []byte) (string, bool) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return "", false
	}

	switch t := v.(type) {
	case string:
		return t + "\n", true
	case []any:
		var b strings.Builder
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return "", false
			}
			b.WriteString(s)
			b.WriteByte('\n')
		}
		return b.String(), true
	default:
		return "", false
	}
}

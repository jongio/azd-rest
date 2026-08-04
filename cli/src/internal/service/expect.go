package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmespath-community/go-jmespath"
	"github.com/jongio/azd-rest/src/internal/client"
)

// expectUsageError signals invalid --expect usage: a non-JSON response, an
// invalid JMESPath expression, or a malformed --expect argument. It reports
// exit code 2 (the invalid-configuration code) through the ExitCoder contract
// so main can tell it apart from an assertion that simply did not hold (exit 1).
type expectUsageError struct{ msg string }

func (e *expectUsageError) Error() string { return e.msg }

// ExitCode returns 2 for invalid --expect usage.
func (e *expectUsageError) ExitCode() int { return 2 }

// evaluateExpectations checks every --expect assertion against the JSON body.
// Each assertion is a JMESPath expression, optionally followed by "=value" to
// require a specific result. A bare expression passes when its result is truthy
// under JMESPath rules. An assertion that does not hold returns a plain error
// (exit 1). A non-JSON body, an empty or invalid expression, or a malformed
// argument returns an expectUsageError (exit 2).
//
// body must be the original response JSON captured before --query rewrites it,
// so --expect asserts on the full response regardless of what --query prints.
func evaluateExpectations(body []byte, contentType string, expects []string) error {
	if len(expects) == 0 {
		return nil
	}
	if !strings.Contains(strings.ToLower(contentType), "json") && !client.IsJSON(body) {
		return &expectUsageError{msg: "--expect requires a JSON response"}
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var data any
	if err := dec.Decode(&data); err != nil {
		return &expectUsageError{msg: fmt.Sprintf("failed to parse JSON response for --expect: %v", err)}
	}

	for _, raw := range expects {
		if err := evaluateExpectation(data, raw); err != nil {
			return err
		}
	}
	return nil
}

// evaluateExpectation runs a single --expect assertion against the decoded data.
func evaluateExpectation(data any, raw string) error {
	expr, expected, hasEquality := splitExpectArg(raw)
	if expr == "" {
		return &expectUsageError{msg: fmt.Sprintf("invalid --expect %q: expression is empty", raw)}
	}

	result, err := jmespath.Search(expr, data)
	if err != nil {
		return &expectUsageError{msg: fmt.Sprintf("invalid --expect expression %q: %v", expr, err)}
	}

	if hasEquality {
		got := stringifyExpectResult(result)
		if got != strings.TrimSpace(expected) {
			return fmt.Errorf("--expect %q failed: expected %q, got %q", raw, strings.TrimSpace(expected), got)
		}
		return nil
	}

	if !isExpectTruthy(result) {
		return fmt.Errorf("--expect %q failed: expression is not truthy (got %s)", expr, stringifyExpectResult(result))
	}
	return nil
}

// splitExpectArg separates a JMESPath expression from an optional "=value"
// equality suffix. It splits on the first standalone "=" so JMESPath
// comparators (==, !=, <=, >=) inside a bare boolean expression are preserved.
// When no standalone "=" is present the whole argument is a truthy expression.
func splitExpectArg(raw string) (expr, expected string, hasEquality bool) {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '=' {
			continue
		}
		// Skip "==" (equality comparator): consume the second '=' too.
		if i+1 < len(raw) && raw[i+1] == '=' {
			i++
			continue
		}
		// Skip "!=", "<=", ">=" (a comparator ending in '=').
		if i > 0 && (raw[i-1] == '!' || raw[i-1] == '<' || raw[i-1] == '>') {
			continue
		}
		return strings.TrimSpace(raw[:i]), raw[i+1:], true
	}
	return strings.TrimSpace(raw), "", false
}

// stringifyExpectResult renders a JMESPath result as a string for equality
// comparison. Scalars render as their plain value so "expr=Succeeded" matches a
// JSON string and "expr=3" matches a number; composite values fall back to their
// compact JSON form.
func stringifyExpectResult(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case json.Number:
		return t.String()
	default:
		// Covers nil (marshals to "null"), numbers, arrays, and objects.
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}

// isExpectTruthy reports whether a JMESPath result is truthy under JMESPath
// rules: false, null, an empty string, an empty array, and an empty object are
// falsy; every other value (including any number) is truthy.
func isExpectTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	default:
		return true
	}
}

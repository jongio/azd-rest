package service

import (
	"fmt"
	"strings"

	"github.com/jongio/azd-rest/src/internal/client"
)

type headerExpectation struct {
	name      string
	value     string
	hasValue  bool
	original  string
	separator string
}

func parseHeaderExpectation(raw string) (headerExpectation, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return headerExpectation{}, fmt.Errorf("--expect-header cannot be empty")
	}

	separatorIndex := -1
	separator := ""
	for _, candidate := range []string{"=", ":"} {
		if idx := strings.Index(text, candidate); idx >= 0 && (separatorIndex == -1 || idx < separatorIndex) {
			separatorIndex = idx
			separator = candidate
		}
	}

	if separatorIndex == -1 {
		return headerExpectation{name: text, original: raw}, nil
	}

	name := strings.TrimSpace(text[:separatorIndex])
	if name == "" {
		return headerExpectation{}, fmt.Errorf("invalid --expect-header %q: header name is required", raw)
	}

	return headerExpectation{
		name:      name,
		value:     strings.TrimSpace(text[separatorIndex+1:]),
		hasValue:  true,
		original:  raw,
		separator: separator,
	}, nil
}

func checkExpectedHeaders(resp *client.Response, expectations []string) error {
	for _, raw := range expectations {
		exp, err := parseHeaderExpectation(raw)
		if err != nil {
			return err
		}

		values := resp.Headers.Values(exp.name)
		if len(values) == 0 {
			return fmt.Errorf("expected response header %q to be present", exp.name)
		}
		if !exp.hasValue {
			continue
		}

		matched := false
		for _, actual := range values {
			if actual == exp.value {
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		return fmt.Errorf(
			"expected response header %q to equal %q, got %s",
			exp.name,
			client.RedactSensitiveHeader(exp.name, exp.value),
			redactedHeaderValues(exp.name, values),
		)
	}
	return nil
}

func redactedHeaderValues(name string, values []string) string {
	if len(values) == 0 {
		return "<none>"
	}
	redacted := make([]string, 0, len(values))
	for _, value := range values {
		redacted = append(redacted, fmt.Sprintf("%q", client.RedactSensitiveHeader(name, value)))
	}
	return strings.Join(redacted, ", ")
}

func validateHeaderExpectations(expectations []string) error {
	for _, raw := range expectations {
		if _, err := parseHeaderExpectation(raw); err != nil {
			return err
		}
	}
	return nil
}

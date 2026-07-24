package service

import (
	"fmt"
	"strconv"
	"strings"
)

type allowedStatusRanges [][2]int

func (r allowedStatusRanges) allows(status int) bool {
	for _, item := range r {
		if status >= item[0] && status <= item[1] {
			return true
		}
	}
	return false
}

func parseAllowedStatuses(spec string) (allowedStatusRanges, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}

	var ranges allowedStatusRanges
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, &allowStatusError{err: fmt.Errorf("--allow-status contains an empty item")}
		}

		startText, endText, isRange := strings.Cut(part, "-")
		start, err := parseHTTPStatus(startText)
		if err != nil {
			return nil, err
		}
		end := start
		if isRange {
			end, err = parseHTTPStatus(endText)
			if err != nil {
				return nil, err
			}
			if end < start {
				return nil, &allowStatusError{err: fmt.Errorf("--allow-status range %q ends before it starts", part)}
			}
		}
		ranges = append(ranges, [2]int{start, end})
	}
	return ranges, nil
}

func parseHTTPStatus(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, &allowStatusError{err: fmt.Errorf("--allow-status contains an empty status code")}
	}
	status, err := strconv.Atoi(value)
	if err != nil {
		return 0, &allowStatusError{err: fmt.Errorf("--allow-status value %q is not a number", value)}
	}
	if status < 100 || status > 599 {
		return 0, &allowStatusError{err: fmt.Errorf("--allow-status value %d is outside 100-599", status)}
	}
	return status, nil
}

type allowStatusError struct{ err error }

// Error returns the invalid --allow-status message.
func (e *allowStatusError) Error() string { return e.err.Error() }

// Unwrap exposes the wrapped error.
func (e *allowStatusError) Unwrap() error { return e.err }

// ExitCode returns 2 to mark invalid flag input as a usage error.
func (e *allowStatusError) ExitCode() int { return 2 }

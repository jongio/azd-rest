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

	parts := strings.Split(spec, ",")
	ranges := make(allowedStatusRanges, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, newAllowStatusError("--allow-status contains an empty item")
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
				return nil, newAllowStatusError(fmt.Sprintf("--allow-status range %q ends before it starts", part))
			}
		}
		ranges = append(ranges, [2]int{start, end})
	}
	return ranges, nil
}

func parseHTTPStatus(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, newAllowStatusError("--allow-status contains an empty status code")
	}
	status, err := strconv.Atoi(value)
	if err != nil {
		return 0, newAllowStatusError(fmt.Sprintf("--allow-status value %q is not a number", value))
	}
	if status < 100 || status > 599 {
		return 0, newAllowStatusError(fmt.Sprintf("--allow-status value %d is outside 100-599", status))
	}
	return status, nil
}


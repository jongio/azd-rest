package service

import (
	"fmt"
	"time"
)

// maxLatencyExitCode is the process exit code returned when a response finishes
// but takes longer than the --max-latency budget. It matches curl's operation
// timeout exit code (28) so scripts can treat a slow response like a timeout.
const maxLatencyExitCode = 28

// maxLatencyConfigError signals that the --max-latency value could not be
// parsed as a positive duration. It reports exit code 2 (invalid configuration)
// through the ExitCoder contract so main exits before any request is made.
type maxLatencyConfigError struct{ value string }

func (e *maxLatencyConfigError) Error() string {
	return fmt.Sprintf("invalid --max-latency %q: use a positive duration such as 500ms or 2s", e.value)
}

// ExitCode returns 2 for an invalid --max-latency value.
func (e *maxLatencyConfigError) ExitCode() int { return 2 }

// maxLatencyExceededError signals that a response completed but its total time
// exceeded the --max-latency budget. The body is written before this error is
// returned, so it only changes the exit code. It reports exit code 28.
type maxLatencyExceededError struct {
	budget time.Duration
	actual time.Duration
}

func (e *maxLatencyExceededError) Error() string {
	return fmt.Sprintf("response took %s, over the --max-latency budget of %s", e.actual, e.budget)
}

// ExitCode returns 28 for a response that overran the latency budget.
func (e *maxLatencyExceededError) ExitCode() int { return maxLatencyExitCode }

// parseMaxLatency turns the raw --max-latency flag value into a duration budget.
// An empty value means the check is disabled and returns a zero budget. A value
// that does not parse, or that is zero or negative, is rejected so a typo never
// silently turns the gate off.
func parseMaxLatency(value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	budget, err := time.ParseDuration(value)
	if err != nil || budget <= 0 {
		return 0, &maxLatencyConfigError{value: value}
	}
	return budget, nil
}

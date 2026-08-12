package service

import "time"

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
		return 0, newMaxLatencyConfigError(value)
	}
	return budget, nil
}

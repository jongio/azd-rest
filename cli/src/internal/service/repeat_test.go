package service

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestPercentile(t *testing.T) {
	sorted := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}

	tests := []struct {
		name string
		p    float64
		want time.Duration
	}{
		{"p50", 50, 30 * time.Millisecond},
		{"p95", 95, 50 * time.Millisecond},
		{"p99", 99, 50 * time.Millisecond},
		{"p0", 0, 10 * time.Millisecond},
		{"p100", 100, 50 * time.Millisecond},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := percentile(sorted, tc.p); got != tc.want {
				t.Errorf("percentile(%v) = %v, want %v", tc.p, got, tc.want)
			}
		})
	}
}

func TestPercentileEmpty(t *testing.T) {
	if got := percentile(nil, 50); got != 0 {
		t.Errorf("percentile(nil) = %v, want 0", got)
	}
}

func TestMeanDuration(t *testing.T) {
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
	}
	if got := meanDuration(durations); got != 20*time.Millisecond {
		t.Errorf("meanDuration = %v, want 20ms", got)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{10 * time.Millisecond, "10.00ms"},
		{1500 * time.Microsecond, "1.50ms"},
		{0, "0.00ms"},
	}
	for _, tc := range tests {
		if got := formatDuration(tc.d); got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestSafeMethods(t *testing.T) {
	safe := []string{"GET", "HEAD", "OPTIONS"}
	for _, m := range safe {
		if !safeMethods[m] {
			t.Errorf("expected %s to be a safe method", m)
		}
	}
	unsafe := []string{"POST", "PUT", "PATCH", "DELETE"}
	for _, m := range unsafe {
		if safeMethods[m] {
			t.Errorf("expected %s not to be a safe method", m)
		}
	}
}

func TestWriteRepeatSummary(t *testing.T) {
	stats := repeatStats{
		total:   3,
		success: 2,
		failed:  1,
		statusCounts: map[int]int{
			200: 2,
			500: 1,
		},
		durations: []time.Duration{
			10 * time.Millisecond,
			20 * time.Millisecond,
			30 * time.Millisecond,
		},
	}

	var buf bytes.Buffer
	writeRepeatSummary(&buf, stats)
	out := buf.String()

	for _, want := range []string{
		"Repeat summary (3 requests)",
		"Success: 2",
		"Failed: 1",
		"200: 2",
		"500: 1",
		"Latency:",
		"min 10.00ms",
		"max 30.00ms",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestWriteRepeatSummaryNoDurations(t *testing.T) {
	stats := repeatStats{
		total:        1,
		success:      0,
		failed:       1,
		statusCounts: map[int]int{},
	}
	var buf bytes.Buffer
	writeRepeatSummary(&buf, stats)
	out := buf.String()
	if strings.Contains(out, "Latency:") {
		t.Errorf("expected no latency line when there are no durations, got:\n%s", out)
	}
}

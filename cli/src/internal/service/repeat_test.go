package service

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWaitRepeatDelayNoDelayReturnsImmediately(t *testing.T) {
	start := time.Now()
	require.NoError(t, waitRepeatDelay(context.Background(), 0))
	assert.Less(t, time.Since(start), 20*time.Millisecond)
}

func TestWaitRepeatDelayWaits(t *testing.T) {
	delay := 20 * time.Millisecond
	start := time.Now()
	require.NoError(t, waitRepeatDelay(context.Background(), delay))
	assert.GreaterOrEqual(t, time.Since(start), delay)
}

func TestWaitRepeatDelayCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitRepeatDelay(ctx, time.Second)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestExecuteRejectsNegativeRepeatDelay(t *testing.T) {
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.RepeatDelay = -time.Millisecond

	err := newTestService().Execute(context.Background(), cfg, "GET", "http://example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--repeat-delay")
}

func TestExecuteRepeatDelaySpacesAttempts(t *testing.T) {
	var seen []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		seen = append(seen, time.Now())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.Repeat = 2
	cfg.RepeatDelay = 25 * time.Millisecond
	cfg.OutputFile = filepath.Join(t.TempDir(), "out.json")

	err := newTestService().Execute(context.Background(), cfg, "GET", server.URL)
	require.NoError(t, err)
	require.Len(t, seen, 2)
	assert.GreaterOrEqual(t, seen[1].Sub(seen[0]), cfg.RepeatDelay)

	out, err := os.ReadFile(cfg.OutputFile)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"ok"`)
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

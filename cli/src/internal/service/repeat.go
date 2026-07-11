package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
)

// safeMethods are HTTP methods that are not expected to change server state.
// Repeating any other method may cause side effects, so we warn about it.
var safeMethods = map[string]bool{
	"GET":     true,
	"HEAD":    true,
	"OPTIONS": true,
}

// repeatStats holds the outcome of a --repeat run.
type repeatStats struct {
	total        int
	success      int
	failed       int
	statusCounts map[int]int
	durations    []time.Duration
}

// executeRepeat sends the same request cfg.Repeat times, collects latency and
// status statistics, prints a summary to stderr, and writes only the last
// successful response body to the configured output.
func (s *RequestService) executeRepeat(ctx context.Context, cfg config.Config, httpClient *client.Client, opts client.RequestOptions) error {
	// Buffer the body so each iteration gets a fresh reader. An io.Reader can
	// only be consumed once, so without this the second request would send an
	// empty body.
	var bodyBytes []byte
	if opts.Body != nil {
		b, err := io.ReadAll(opts.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}
		bodyBytes = b
	}

	if !safeMethods[opts.Method] {
		fmt.Fprintf(os.Stderr, "Warning: repeating a %s request %d times may cause side effects.\n", opts.Method, cfg.Repeat)
	}

	stats := repeatStats{
		total:        cfg.Repeat,
		statusCounts: make(map[int]int),
		durations:    make([]time.Duration, 0, cfg.Repeat),
	}

	var lastResp *client.Response
	for i := 0; i < cfg.Repeat; i++ {
		if bodyBytes != nil {
			opts.Body = bytes.NewReader(bodyBytes)
		}

		resp, err := httpClient.Execute(ctx, opts)
		if err != nil {
			stats.failed++
			fmt.Fprintf(os.Stderr, "Request %d/%d failed: %v\n", i+1, cfg.Repeat, err)
			continue
		}

		stats.durations = append(stats.durations, resp.Duration)
		stats.statusCounts[resp.StatusCode]++
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			stats.success++
		} else {
			stats.failed++
		}
		lastResp = resp
	}

	writeRepeatSummary(os.Stderr, stats)

	if lastResp == nil {
		return fmt.Errorf("all %d requests failed", cfg.Repeat)
	}

	if err := writeResponseMetadata(cfg.MetadataFile, opts.Method, opts.URL, lastResp); err != nil {
		return err
	}

	return s.writeResponseOutput(cfg, lastResp)
}

// writeRepeatSummary prints the repeat run statistics to w.
func writeRepeatSummary(w io.Writer, stats repeatStats) {
	fmt.Fprintf(w, "\nRepeat summary (%d requests):\n", stats.total)
	fmt.Fprintf(w, "  Success: %d   Failed: %d\n", stats.success, stats.failed)

	if len(stats.statusCounts) > 0 {
		codes := make([]int, 0, len(stats.statusCounts))
		for code := range stats.statusCounts {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		fmt.Fprintf(w, "  Status:")
		for _, code := range codes {
			fmt.Fprintf(w, "  %d: %d", code, stats.statusCounts[code])
		}
		fmt.Fprintln(w)
	}

	if len(stats.durations) > 0 {
		sorted := make([]time.Duration, len(stats.durations))
		copy(sorted, stats.durations)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		fmt.Fprintf(w, "  Latency: min %s  mean %s  p50 %s  p95 %s  p99 %s  max %s\n",
			formatDuration(sorted[0]),
			formatDuration(meanDuration(sorted)),
			formatDuration(percentile(sorted, 50)),
			formatDuration(percentile(sorted, 95)),
			formatDuration(percentile(sorted, 99)),
			formatDuration(sorted[len(sorted)-1]),
		)
	}
}

// meanDuration returns the arithmetic mean of the durations.
func meanDuration(durations []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

// percentile returns the p-th percentile (0-100) of a sorted duration slice
// using the nearest-rank method.
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	rank := int((p/100.0)*float64(len(sorted)) + 0.5)
	if rank < 1 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}

// formatDuration renders a duration in milliseconds with two decimals.
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.2fms", float64(d)/float64(time.Millisecond))
}

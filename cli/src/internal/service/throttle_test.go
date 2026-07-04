package service

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestWriteThrottleInfo_Present(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Ms-Ratelimit-Remaining-Subscription-Reads", "11999")
	headers.Set("X-Ms-Ratelimit-Remaining-Tenant-Writes", "2999")
	headers.Set("Content-Type", "application/json")

	var buf bytes.Buffer
	writeThrottleInfo(&buf, headers)
	out := buf.String()

	for _, want := range []string{
		"Rate limit / quota:",
		"x-ms-ratelimit-remaining-subscription-reads: 11999",
		"x-ms-ratelimit-remaining-tenant-writes: 2999",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Warning") {
		t.Errorf("did not expect a low-quota warning\ngot:\n%s", out)
	}
	if strings.Contains(out, "Content-Type") {
		t.Errorf("unrelated headers should not appear\ngot:\n%s", out)
	}
}

func TestWriteThrottleInfo_LowQuotaWarning(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Ms-Ratelimit-Remaining-Subscription-Writes", "5")

	var buf bytes.Buffer
	writeThrottleInfo(&buf, headers)
	out := buf.String()

	if !strings.Contains(out, "x-ms-ratelimit-remaining-subscription-writes: 5") {
		t.Errorf("missing header line\ngot:\n%s", out)
	}
	if !strings.Contains(out, "Warning: low remaining quota for x-ms-ratelimit-remaining-subscription-writes (5).") {
		t.Errorf("missing low-quota warning\ngot:\n%s", out)
	}
}

func TestWriteThrottleInfo_NonIntegerValueNoWarning(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Ms-Ratelimit-Remaining-Resource", "Microsoft.Compute/HighCostGet3Min;107")

	var buf bytes.Buffer
	writeThrottleInfo(&buf, headers)
	out := buf.String()

	if !strings.Contains(out, "x-ms-ratelimit-remaining-resource: Microsoft.Compute/HighCostGet3Min;107") {
		t.Errorf("missing header line\ngot:\n%s", out)
	}
	if strings.Contains(out, "Warning") {
		t.Errorf("non-integer values must not trigger a warning\ngot:\n%s", out)
	}
}

func TestWriteThrottleInfo_Absent(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Request-Id", "abc")

	var buf bytes.Buffer
	writeThrottleInfo(&buf, headers)

	if buf.Len() != 0 {
		t.Errorf("expected no output when no throttle headers present, got:\n%s", buf.String())
	}
}

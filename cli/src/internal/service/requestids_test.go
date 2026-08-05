package service

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestWriteRequestIDs_PrintsKnownHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Ms-Request-Id", "req-1")
	headers.Set("X-Ms-Correlation-Request-Id", "corr-1")
	headers.Set("X-Ms-Routing-Request-Id", "route-1")
	headers.Set("Request-Id", "trace-1")
	headers.Set("Client-Request-Id", "client-1")
	headers.Set("X-Other", "ignored")

	var buf bytes.Buffer
	writeRequestIDs(&buf, headers)
	got := buf.String()

	for _, want := range []string{
		"x-ms-request-id: req-1",
		"x-ms-correlation-request-id: corr-1",
		"x-ms-routing-request-id: route-1",
		"request-id: trace-1",
		"client-request-id: client-1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in output:\n%s", want, got)
		}
	}
	if strings.Contains(got, "ignored") {
		t.Fatalf("unexpected non-request ID header in output:\n%s", got)
	}
}

func TestWriteRequestIDs_NoKnownHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Other", "ignored")

	var buf bytes.Buffer
	writeRequestIDs(&buf, headers)

	if got := buf.String(); got != "" {
		t.Fatalf("expected no output, got %q", got)
	}
}

package service

import (
	"bytes"
	"strings"
	"testing"
)

// TestWriteDiagnostic_NotSilent verifies advisory messages are written when
// silent mode is off.
func TestWriteDiagnostic_NotSilent(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostic(&buf, false, "Warning: %s\n", "disabled")

	got := buf.String()
	if !strings.Contains(got, "Warning: disabled") {
		t.Fatalf("expected diagnostic to be written, got %q", got)
	}
}

// TestWriteDiagnostic_Silent verifies advisory messages are suppressed when
// silent mode is on.
func TestWriteDiagnostic_Silent(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostic(&buf, true, "Warning: %s\n", "disabled")

	if got := buf.String(); got != "" {
		t.Fatalf("expected no diagnostic output in silent mode, got %q", got)
	}
}

// TestWriteDiagnostic_FormatsArgs verifies the helper formats arguments like
// fmt.Fprintf when not silent.
func TestWriteDiagnostic_FormatsArgs(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostic(&buf, false, "> Pagination enabled (max %d pages)\n", 100)

	if got := buf.String(); got != "> Pagination enabled (max 100 pages)\n" {
		t.Fatalf("unexpected formatted output: %q", got)
	}
}

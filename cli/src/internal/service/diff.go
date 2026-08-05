package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/pmezard/go-difflib/difflib"
)

// diffUsageError signals invalid --diff usage: a missing or unreadable baseline
// file, a baseline that is not JSON, or a non-JSON response. It reports exit
// code 2 through the ExitCoder contract so main can tell it apart from a drift
// result, which is a plain error (exit 1).
type diffUsageError struct{ msg string }

func (e *diffUsageError) Error() string { return e.msg }

// ExitCode returns 2 for invalid --diff usage.
func (e *diffUsageError) ExitCode() int { return 2 }

// diffAgainstBaseline compares the JSON response against a saved baseline file.
// Both sides are canonicalized (parsed as JSON and re-encoded with sorted keys
// and indentation) so key order and whitespace do not produce false diffs.
//
// When they match it prints nothing and returns nil. When they differ it writes
// a unified diff to out and returns a plain error so the command exits non-zero.
// A missing or unreadable baseline, a non-JSON baseline, or a non-JSON response
// returns a diffUsageError (exit 2).
func diffAgainstBaseline(out io.Writer, body []byte, baselinePath string) error {
	baselineRaw, err := os.ReadFile(baselinePath) // #nosec G304 -- User-specified baseline path via --diff flag is intentional.
	if err != nil {
		return &diffUsageError{msg: fmt.Sprintf("failed to read --diff baseline %q: %v", baselinePath, err)}
	}

	if !client.IsJSON(body) {
		return &diffUsageError{msg: "--diff requires a JSON response"}
	}

	responseCanon, err := canonicalizeJSON(body)
	if err != nil {
		return &diffUsageError{msg: fmt.Sprintf("failed to parse JSON response for --diff: %v", err)}
	}
	baselineCanon, err := canonicalizeJSON(baselineRaw)
	if err != nil {
		return &diffUsageError{msg: fmt.Sprintf("--diff baseline %q is not valid JSON: %v", baselinePath, err)}
	}

	if bytes.Equal(responseCanon, baselineCanon) {
		return nil
	}

	diffText, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(baselineCanon) + "\n"),
		B:        difflib.SplitLines(string(responseCanon) + "\n"),
		FromFile: "baseline",
		ToFile:   "response",
		Context:  3,
	})
	if err != nil {
		return fmt.Errorf("failed to render --diff output: %w", err)
	}

	fmt.Fprint(out, diffText)
	return fmt.Errorf("response differs from --diff baseline %q", baselinePath)
}

// canonicalizeJSON parses raw as JSON and re-encodes it with sorted object keys
// and two-space indentation so two documents that differ only in key order or
// whitespace produce identical output. Numbers keep their original text.
func canonicalizeJSON(raw []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

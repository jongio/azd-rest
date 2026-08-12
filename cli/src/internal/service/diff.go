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

// diffAgainstBaseline compares the JSON response against a saved baseline file.
// Both sides are canonicalized (parsed as JSON and re-encoded with sorted keys
// and indentation) so key order and whitespace do not produce false diffs.
//
// When they match it prints nothing and returns nil. When they differ it writes
// a unified diff to out and returns a plain error so the command exits non-zero.
// A missing or unreadable baseline, a non-JSON baseline, or a non-JSON response
// is invalid usage and returns a structured usage error instead.
func diffAgainstBaseline(out io.Writer, body []byte, baselinePath string) error {
	baselineRaw, err := os.ReadFile(baselinePath) // #nosec G304 -- User-specified baseline path via --diff flag is intentional.
	if err != nil {
		return newDiffUsageError(fmt.Sprintf("failed to read --diff baseline %q: %v", baselinePath, err))
	}

	if !client.IsJSON(body) {
		return newDiffUsageError("--diff requires a JSON response")
	}

	responseCanon, err := canonicalizeJSON(body)
	if err != nil {
		return newDiffUsageError(fmt.Sprintf("failed to parse JSON response for --diff: %v", err))
	}
	baselineCanon, err := canonicalizeJSON(baselineRaw)
	if err != nil {
		return newDiffUsageError(fmt.Sprintf("--diff baseline %q is not valid JSON: %v", baselinePath, err))
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

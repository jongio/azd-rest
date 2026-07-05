package service

import (
	"encoding/csv"
	"strings"
	"testing"
)

// parseCSV reads the rendered CSV back into records so tests assert on decoded
// values rather than raw quoting, then also returns the raw text for exact
// checks where quoting matters.
func parseCSV(t *testing.T, out string) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v\n%s", err, out)
	}
	return records
}

func TestRenderCSVTopLevelArray(t *testing.T) {
	body := []byte(`[
		{"name":"alpha","location":"eastus2","tier":"free"},
		{"name":"beta","location":"eastus","tier":"paid"}
	]`)

	out, err := renderCSV(body)
	if err != nil {
		t.Fatalf("renderCSV returned error: %v", err)
	}

	records := parseCSV(t, out)
	if len(records) != 3 {
		t.Fatalf("expected header + 2 rows, got %d records: %v", len(records), records)
	}

	// name and location are priority columns and come first, in that order;
	// remaining keys (tier) follow alphabetically.
	wantHeader := []string{"name", "location", "tier"}
	if strings.Join(records[0], ",") != strings.Join(wantHeader, ",") {
		t.Errorf("header = %v, want %v", records[0], wantHeader)
	}
	if strings.Join(records[1], ",") != "alpha,eastus2,free" {
		t.Errorf("row 1 = %v", records[1])
	}
	if strings.Join(records[2], ",") != "beta,eastus,paid" {
		t.Errorf("row 2 = %v", records[2])
	}
}

func TestRenderCSVWrappedInValue(t *testing.T) {
	body := []byte(`{"value":[{"name":"alpha"},{"name":"beta"}]}`)

	out, err := renderCSV(body)
	if err != nil {
		t.Fatalf("renderCSV returned error: %v", err)
	}

	records := parseCSV(t, out)
	if len(records) != 3 {
		t.Fatalf("expected header + 2 rows, got %d: %v", len(records), records)
	}
	if records[0][0] != "name" {
		t.Errorf("header = %v, want [name]", records[0])
	}
	if records[1][0] != "alpha" || records[2][0] != "beta" {
		t.Errorf("rows = %v, %v", records[1], records[2])
	}
}

func TestRenderCSVQuoting(t *testing.T) {
	// Values containing commas, quotes, and newlines must survive a round trip.
	body := []byte(`[{"name":"has, comma","note":"has \"quote\"","detail":"line one\nline two"}]`)

	out, err := renderCSV(body)
	if err != nil {
		t.Fatalf("renderCSV returned error: %v", err)
	}

	records := parseCSV(t, out)
	if len(records) != 2 {
		t.Fatalf("expected header + 1 row, got %d: %v", len(records), records)
	}

	// Header order: name (priority) first, then detail, note (alphabetical).
	got := map[string]string{}
	for i, col := range records[0] {
		got[col] = records[1][i]
	}
	if got["name"] != "has, comma" {
		t.Errorf("name = %q, want %q", got["name"], "has, comma")
	}
	if got["note"] != `has "quote"` {
		t.Errorf("note = %q, want %q", got["note"], `has "quote"`)
	}
	// Newlines in a cell are collapsed to spaces so each row stays on one line.
	if got["detail"] != "line one line two" {
		t.Errorf("detail = %q, want %q", got["detail"], "line one line two")
	}

	// The comma-bearing value must be wrapped in quotes in the raw output.
	if !strings.Contains(out, `"has, comma"`) {
		t.Errorf("expected quoted comma value in raw output:\n%s", out)
	}
}

func TestRenderCSVNestedValues(t *testing.T) {
	body := []byte(`[{"name":"alpha","tags":{"tier":"free"},"zones":["one","two"]}]`)

	out, err := renderCSV(body)
	if err != nil {
		t.Fatalf("renderCSV returned error: %v", err)
	}

	records := parseCSV(t, out)
	if len(records) != 2 {
		t.Fatalf("expected header + 1 row, got %d: %v", len(records), records)
	}

	got := map[string]string{}
	for i, col := range records[0] {
		got[col] = records[1][i]
	}
	if got["tags"] != `{"tier":"free"}` {
		t.Errorf("tags = %q, want compact JSON object", got["tags"])
	}
	if got["zones"] != `["one","two"]` {
		t.Errorf("zones = %q, want compact JSON array", got["zones"])
	}
}

func TestRenderCSVScalarRows(t *testing.T) {
	body := []byte(`["alpha","beta","gamma"]`)

	out, err := renderCSV(body)
	if err != nil {
		t.Fatalf("renderCSV returned error: %v", err)
	}

	records := parseCSV(t, out)
	if len(records) != 4 {
		t.Fatalf("expected header + 3 rows, got %d: %v", len(records), records)
	}
	if records[0][0] != "value" {
		t.Errorf("header = %v, want [value]", records[0])
	}
	if records[1][0] != "alpha" || records[3][0] != "gamma" {
		t.Errorf("rows = %v", records[1:])
	}
}

func TestRenderCSVEmptyArray(t *testing.T) {
	out, err := renderCSV([]byte(`[]`))
	if err != nil {
		t.Fatalf("renderCSV returned error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output for empty array, got %q", out)
	}
}

func TestRenderCSVInvalidJSON(t *testing.T) {
	_, err := renderCSV([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "csv format requires a JSON response") {
		t.Errorf("error = %q, want csv format message", err.Error())
	}
}

func TestRenderCSVNumbersPreserved(t *testing.T) {
	// UseNumber keeps large integers and precise decimals intact.
	body := []byte(`[{"name":"alpha","count":10000000000,"ratio":0.125}]`)

	out, err := renderCSV(body)
	if err != nil {
		t.Fatalf("renderCSV returned error: %v", err)
	}

	records := parseCSV(t, out)
	got := map[string]string{}
	for i, col := range records[0] {
		got[col] = records[1][i]
	}
	if got["count"] != "10000000000" {
		t.Errorf("count = %q, want 10000000000", got["count"])
	}
	if got["ratio"] != "0.125" {
		t.Errorf("ratio = %q, want 0.125", got["ratio"])
	}
}

package service

import (
	"encoding/csv"
	"strings"
	"testing"
)

func parseCSV(t *testing.T, s string) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(s)).ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v\n%s", err, s)
	}
	return records
}

func TestRenderCSVObjectArray(t *testing.T) {
	body := `[{"name":"a","location":"eastus"},{"name":"b","location":"westus"}]`
	out, err := renderCSV([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := parseCSV(t, out)
	if len(rows) != 3 {
		t.Fatalf("expected header + 2 rows, got %d:\n%s", len(rows), out)
	}
	// name and location are both priority columns; name comes first.
	if rows[0][0] != "name" || rows[0][1] != "location" {
		t.Errorf("unexpected header order: %v", rows[0])
	}
	if rows[1][0] != "a" || rows[1][1] != "eastus" {
		t.Errorf("unexpected first data row: %v", rows[1])
	}
}

func TestRenderCSVWrappedInValue(t *testing.T) {
	body := `{"value":[{"id":"1"},{"id":"2"}],"nextLink":"..."}`
	out, err := renderCSV([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := parseCSV(t, out)
	if len(rows) != 3 {
		t.Fatalf("expected header + 2 rows from value, got %d:\n%s", len(rows), out)
	}
	if rows[0][0] != "id" {
		t.Errorf("expected id header, got %v", rows[0])
	}
}

func TestRenderCSVScalarRows(t *testing.T) {
	body := `["one","two","three"]`
	out, err := renderCSV([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := parseCSV(t, out)
	if len(rows) != 4 {
		t.Fatalf("expected header + 3 rows, got %d:\n%s", len(rows), out)
	}
	if rows[0][0] != "value" {
		t.Errorf("expected single value column, got %v", rows[0])
	}
	if rows[1][0] != "one" {
		t.Errorf("unexpected scalar row: %v", rows[1])
	}
}

func TestRenderCSVQuotesCommasAndQuotes(t *testing.T) {
	body := `[{"name":"a,b","note":"say \"hi\""}]`
	out, err := renderCSV([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Round-trip through the CSV reader to confirm values survive quoting.
	rows := parseCSV(t, out)
	if len(rows) != 2 {
		t.Fatalf("expected header + 1 row, got %d:\n%s", len(rows), out)
	}
	got := map[string]string{}
	for i, col := range rows[0] {
		got[col] = rows[1][i]
	}
	if got["name"] != "a,b" {
		t.Errorf("comma value not preserved: %q", got["name"])
	}
	if got["note"] != `say "hi"` {
		t.Errorf("quote value not preserved: %q", got["note"])
	}
}

func TestRenderCSVNestedObjectAsJSON(t *testing.T) {
	body := `[{"name":"a","sku":{"tier":"Standard"}}]`
	out, err := renderCSV([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := parseCSV(t, out)
	got := map[string]string{}
	for i, col := range rows[0] {
		got[col] = rows[1][i]
	}
	if got["sku"] != `{"tier":"Standard"}` {
		t.Errorf("nested object should be compact JSON, got %q", got["sku"])
	}
}

func TestRenderCSVEmptyArray(t *testing.T) {
	out, err := renderCSV([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output for empty array, got %q", out)
	}
}

func TestRenderCSVInvalidJSON(t *testing.T) {
	if _, err := renderCSV([]byte(`not json`)); err == nil {
		t.Fatalf("expected error for non-JSON body")
	}
}

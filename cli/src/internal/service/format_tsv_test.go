package service

import (
	"encoding/csv"
	"strings"
	"testing"
)

func parseTSV(t *testing.T, s string) [][]string {
	t.Helper()
	r := csv.NewReader(strings.NewReader(s))
	r.Comma = '\t'
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("output is not valid TSV: %v\n%s", err, s)
	}
	return records
}

func TestRenderTSVObjectArray(t *testing.T) {
	body := `[{"name":"a","location":"eastus"},{"name":"b","location":"westus"}]`
	out, err := renderTSV([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := parseTSV(t, out)
	if len(rows) != 3 {
		t.Fatalf("expected header + 2 rows, got %d:\n%s", len(rows), out)
	}
	if rows[0][0] != "name" || rows[0][1] != "location" {
		t.Errorf("unexpected header order: %v", rows[0])
	}
	if rows[1][0] != "a" || rows[1][1] != "eastus" {
		t.Errorf("unexpected first data row: %v", rows[1])
	}
}

func TestRenderTSVWrappedInValue(t *testing.T) {
	body := `{"value":[{"id":"1"},{"id":"2"}],"nextLink":"..."}`
	out, err := renderTSV([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := parseTSV(t, out)
	if len(rows) != 3 {
		t.Fatalf("expected header + 2 rows from value, got %d:\n%s", len(rows), out)
	}
	if rows[0][0] != "id" {
		t.Errorf("expected id header, got %v", rows[0])
	}
}

func TestRenderTSVScalarRows(t *testing.T) {
	body := `["one","two","three"]`
	out, err := renderTSV([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := parseTSV(t, out)
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

func TestRenderTSVQuotesTabs(t *testing.T) {
	body := `[{"name":"a\tb","note":"say \"hi\""}]`
	out, err := renderTSV([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := parseTSV(t, out)
	if len(rows) != 2 {
		t.Fatalf("expected header + 1 row, got %d:\n%s", len(rows), out)
	}
	got := map[string]string{}
	for i, col := range rows[0] {
		got[col] = rows[1][i]
	}
	if got["name"] != "a\tb" {
		t.Errorf("tab value not preserved: %q", got["name"])
	}
	if got["note"] != `say "hi"` {
		t.Errorf("quote value not preserved: %q", got["note"])
	}
}

func TestRenderTSVNestedObjectAsJSON(t *testing.T) {
	body := `[{"name":"a","sku":{"tier":"Standard"}}]`
	out, err := renderTSV([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows := parseTSV(t, out)
	got := map[string]string{}
	for i, col := range rows[0] {
		got[col] = rows[1][i]
	}
	if got["sku"] != `{"tier":"Standard"}` {
		t.Errorf("nested object should be compact JSON, got %q", got["sku"])
	}
}

func TestRenderTSVEmptyArray(t *testing.T) {
	out, err := renderTSV([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output for empty array, got %q", out)
	}
}

func TestRenderTSVInvalidJSON(t *testing.T) {
	if _, err := renderTSV([]byte(`not json`)); err == nil {
		t.Fatalf("expected error for non-JSON body")
	}
}

package service

import (
	"strings"
	"testing"
)

func TestRenderTableTopLevelArray(t *testing.T) {
	body := `[{"name":"vm1","location":"eastus"},{"name":"vm2","location":"eastus2"}]`
	out, err := renderTable([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := splitLines(out)
	if len(lines) != 4 {
		t.Fatalf("expected header + separator + 2 rows, got %d lines:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "NAME") {
		t.Errorf("expected NAME first (priority column), got header: %q", lines[0])
	}
	if !strings.Contains(lines[0], "LOCATION") {
		t.Errorf("expected LOCATION column, got: %q", lines[0])
	}
	if !strings.Contains(out, "vm1") || !strings.Contains(out, "eastus2") {
		t.Errorf("missing row data:\n%s", out)
	}
}

func TestRenderTableWrappedInValue(t *testing.T) {
	body := `{"value":[{"id":"a","type":"t1"},{"id":"b","type":"t2"}]}`
	out, err := renderTable([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "TYPE") {
		t.Errorf("expected ID and TYPE columns:\n%s", out)
	}
	if !strings.Contains(out, "a") || !strings.Contains(out, "t2") {
		t.Errorf("missing row data:\n%s", out)
	}
}

func TestRenderTableResourceGraphData(t *testing.T) {
	// Resource Graph wraps rows under "data".
	body := `{"totalRecords":1,"data":[{"name":"rg1","resourceGroup":"prod"}]}`
	out, err := renderTable([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "RESOURCE") {
		t.Errorf("expected columns from data[] rows:\n%s", out)
	}
}

func TestRenderTableSingleObject(t *testing.T) {
	body := `{"name":"solo","enabled":true}`
	out, err := renderTable([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := splitLines(out)
	if len(lines) != 3 {
		t.Fatalf("expected header + separator + 1 row, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(out, "true") {
		t.Errorf("expected bool rendered as true:\n%s", out)
	}
}

func TestRenderTableScalarArray(t *testing.T) {
	body := `["one","two","three"]`
	out, err := renderTable([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, "VALUE") {
		t.Errorf("expected single VALUE column:\n%s", out)
	}
	if !strings.Contains(out, "two") {
		t.Errorf("missing scalar row:\n%s", out)
	}
}

func TestRenderTableNestedAndNumbers(t *testing.T) {
	body := `[{"name":"x","count":1000,"tags":{"env":"prod"}}]`
	out, err := renderTable([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "1000") {
		t.Errorf("expected integer 1000 without exponent:\n%s", out)
	}
	if !strings.Contains(out, `{"env":"prod"}`) {
		t.Errorf("expected nested object as compact JSON:\n%s", out)
	}
}

func TestRenderTableEmptyArray(t *testing.T) {
	out, err := renderTable([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No results") {
		t.Errorf("expected no-results message, got: %q", out)
	}
}

func TestRenderTableInvalidJSON(t *testing.T) {
	if _, err := renderTable([]byte(`not json`)); err == nil {
		t.Fatalf("expected error for non-JSON body")
	}
}

func splitLines(s string) []string {
	trimmed := strings.TrimRight(s, "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

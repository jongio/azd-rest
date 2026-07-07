package service

import (
	"strings"
	"testing"
)

func TestRenderTableWithColumns_SelectionAndOrder(t *testing.T) {
	body := `[{"name":"vm1","location":"eastus","provisioningState":"Succeeded"},{"name":"vm2","location":"eastus2","provisioningState":"Failed"}]`
	out, err := renderTableWithColumns([]byte(body), []string{"location", "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := splitLines(out)
	if lines[0] != "LOCATION  NAME" {
		t.Errorf("expected only LOCATION then NAME, got header: %q", lines[0])
	}
	if strings.Contains(out, "Succeeded") {
		t.Errorf("unselected column should be omitted:\n%s", out)
	}
	// Column order follows the request: location value before name value.
	if !strings.HasPrefix(lines[2], "eastus") {
		t.Errorf("expected location value first in row, got: %q", lines[2])
	}
}

func TestRenderTableWithColumns_MissingColumnEmptyCell(t *testing.T) {
	body := `[{"name":"vm1","location":"eastus"}]`
	out, err := renderTableWithColumns([]byte(body), []string{"name", "sku"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := splitLines(out)
	if !strings.Contains(lines[0], "SKU") {
		t.Errorf("expected requested SKU column in header even when absent: %q", lines[0])
	}
	// The row keeps the name value; the missing sku cell is trimmed to empty.
	if strings.TrimSpace(lines[2]) != "vm1" {
		t.Errorf("expected only the name value with an empty sku cell, got: %q", lines[2])
	}
}

func TestRenderTableWithColumns_TrimsAndIgnoresBlanks(t *testing.T) {
	body := `[{"name":"vm1","location":"eastus"}]`
	out, err := renderTableWithColumns([]byte(body), []string{" name ", "", "location"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if splitLines(out)[0] != "NAME  LOCATION" {
		t.Errorf("expected trimmed NAME and LOCATION header, got: %q", splitLines(out)[0])
	}
}

func TestRenderTableWithColumns_EmptySelectionUsesAutoLayout(t *testing.T) {
	body := `[{"name":"vm1","location":"eastus"}]`
	auto, err := renderTable([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	withEmpty, err := renderTableWithColumns([]byte(body), []string{"  ", ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if withEmpty != auto {
		t.Errorf("blank-only selection should fall back to auto layout:\ngot:\n%s\nwant:\n%s", withEmpty, auto)
	}
}

func TestRenderTableWithColumns_ScalarRowsIgnoreSelection(t *testing.T) {
	body := `["a","b","c"]`
	withCols, err := renderTableWithColumns([]byte(body), []string{"name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(splitLines(withCols)[0], "VALUE") {
		t.Errorf("scalar rows should still use the single VALUE column, got: %q", splitLines(withCols)[0])
	}
}

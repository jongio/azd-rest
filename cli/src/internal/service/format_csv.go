package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

// renderCSV renders a JSON response body as CSV with a header row, so Azure
// resource listings drop straight into a spreadsheet without a jq step.
//
// It reuses the table renderer's row extraction and column ordering: a
// top-level array, or an object wrapping rows under value, data, results, or
// items, becomes one CSV row per item. Rows that are objects share a column per
// field; scalar rows use a single "value" column. Nested objects and arrays
// render as compact JSON in the cell, matching the table renderer. Go's
// encoding/csv handles quoting and escaping per RFC 4180.
func renderCSV(body []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return "", fmt.Errorf("csv format requires a JSON response: %w", err)
	}

	rows := extractTableRows(parsed)
	if len(rows) == 0 {
		return "", nil
	}

	header, data := buildTableData(rows)

	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}
	for _, record := range data {
		if err := w.Write(record); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("failed to encode CSV: %w", err)
	}
	return b.String(), nil
}

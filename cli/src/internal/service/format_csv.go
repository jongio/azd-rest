package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

// renderCSV renders a JSON response body as RFC 4180 CSV.
//
// It accepts a top-level JSON array, an object that wraps rows under a common
// key (value, data, results, or items), or a single object (rendered as one
// row). Rows that are objects become columns following the same ordering as the
// table format; rows that are scalars render under a single "value" column.
// Nested objects and arrays are written as compact JSON inside the cell. An
// empty result set produces no output.
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

	var b strings.Builder
	w := csv.NewWriter(&b)

	columns, allObjects := tableColumns(rows)
	if allObjects {
		if err := writeCSVRecord(w, columns); err != nil {
			return "", err
		}
		for _, row := range rows {
			obj, _ := row.(map[string]any)
			cells := make([]string, len(columns))
			for i, col := range columns {
				cells[i] = tableCellString(obj[col])
			}
			if err := writeCSVRecord(w, cells); err != nil {
				return "", err
			}
		}
	} else {
		if err := writeCSVRecord(w, []string{valueKey}); err != nil {
			return "", err
		}
		for _, row := range rows {
			if err := writeCSVRecord(w, []string{tableCellString(row)}); err != nil {
				return "", err
			}
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("failed to write csv: %w", err)
	}
	return b.String(), nil
}

// writeCSVRecord writes a single CSV record and wraps any error with context.
func writeCSVRecord(w *csv.Writer, record []string) error {
	if err := w.Write(record); err != nil {
		return fmt.Errorf("failed to write csv record: %w", err)
	}
	return nil
}

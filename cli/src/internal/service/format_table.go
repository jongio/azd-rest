package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// tableColumnPriority lists keys that are shown first (in this order) when a
// row object contains them. Remaining keys follow in alphabetical order. This
// gives resource listings a predictable, readable column layout.
var tableColumnPriority = []string{
	"name", "id", "type", "location", "resourceGroup",
	"subscriptionId", "kind", "status", "provisioningState",
}

// renderTable renders a JSON response body as an aligned text table using the
// automatic column layout.
func renderTable(body []byte) (string, error) {
	return renderTableWithColumns(body, nil)
}

// renderTableWithColumns renders a JSON response body as an aligned text table.
//
// It accepts a top-level JSON array, or an object that wraps rows under a
// common key (value, data, results, or items), or a single object (rendered as
// one row). Rows that are objects become columns; rows that are scalars render
// under a single "value" column.
//
// When selected is non-empty and every row is an object, only those columns are
// rendered, in the given order, instead of the automatic layout. A selected
// column that is missing from a row renders as an empty cell. Blank names are
// ignored; if nothing usable remains, the automatic layout is used.
func renderTableWithColumns(body []byte, selected []string) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return "", fmt.Errorf("table format requires a JSON response: %w", err)
	}

	rows := extractTableRows(parsed)
	if len(rows) == 0 {
		return "No results.\n", nil
	}

	columns, allObjects := tableColumns(rows)

	var header []string
	var data [][]string
	if allObjects {
		if cleaned := cleanColumnNames(selected); len(cleaned) > 0 {
			columns = cleaned
		}
		header = columns
		for _, row := range rows {
			obj, _ := row.(map[string]any)
			cells := make([]string, len(columns))
			for i, col := range columns {
				cells[i] = tableCellString(obj[col])
			}
			data = append(data, cells)
		}
	} else {
		header = []string{"value"}
		for _, row := range rows {
			data = append(data, []string{tableCellString(row)})
		}
	}

	return formatTable(header, data), nil
}

// cleanColumnNames trims surrounding spaces from requested column names and
// drops empty entries so "--table-columns name, location" behaves the same as
// "--table-columns name,location".
func cleanColumnNames(selected []string) []string {
	cleaned := make([]string, 0, len(selected))
	for _, col := range selected {
		if trimmed := strings.TrimSpace(col); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

// extractTableRows normalizes a parsed JSON value into a slice of row values.
func extractTableRows(parsed any) []any {
	switch v := parsed.(type) {
	case []any:
		return v
	case map[string]any:
		for _, key := range []string{"value", "data", "results", "items"} {
			if arr, ok := v[key].([]any); ok {
				return arr
			}
		}
		return []any{v}
	default:
		return []any{parsed}
	}
}

// tableColumns returns the ordered column names across all row objects and
// reports whether every row is an object. When any row is a scalar, the caller
// renders a single "value" column instead.
func tableColumns(rows []any) ([]string, bool) {
	seen := make(map[string]bool)
	var extra []string
	allObjects := true

	for _, row := range rows {
		obj, ok := row.(map[string]any)
		if !ok {
			allObjects = false
			continue
		}
		for key := range obj {
			if !seen[key] {
				seen[key] = true
				extra = append(extra, key)
			}
		}
	}

	if !allObjects {
		return nil, false
	}

	var ordered []string
	priority := make(map[string]bool)
	for _, key := range tableColumnPriority {
		if seen[key] {
			ordered = append(ordered, key)
			priority[key] = true
		}
	}

	var rest []string
	for _, key := range extra {
		if !priority[key] {
			rest = append(rest, key)
		}
	}
	sortStrings(rest)

	return append(ordered, rest...), true
}

// tableCellString renders a single JSON value as a table cell. Nested objects
// and arrays are shown as compact JSON. Newlines are collapsed so each row
// stays on one line.
func tableCellString(v any) string {
	var s string
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		s = t
	case bool:
		s = strconv.FormatBool(t)
	case json.Number:
		s = t.String()
	case float64:
		s = strconv.FormatFloat(t, 'f', -1, 64)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			s = fmt.Sprintf("%v", t)
		} else {
			s = string(b)
		}
	}
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// formatTable lays out the header and rows as a fixed-width text table with an
// uppercase header, a dashed separator, and two spaces between columns.
func formatTable(header []string, rows [][]string) string {
	widths := make([]int, len(header))
	upper := make([]string, len(header))
	for i, h := range header {
		upper[i] = strings.ToUpper(h)
		widths[i] = len(upper[i])
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var b strings.Builder
	writeTableLine(&b, upper, widths)

	separators := make([]string, len(header))
	for i, w := range widths {
		separators[i] = strings.Repeat("-", w)
	}
	writeTableLine(&b, separators, widths)

	for _, row := range rows {
		writeTableLine(&b, row, widths)
	}
	return b.String()
}

// writeTableLine writes one padded, right-trimmed table row.
func writeTableLine(b *strings.Builder, cells []string, widths []int) {
	var parts []string
	for i, w := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		if i == len(widths)-1 {
			parts = append(parts, cell)
		} else {
			parts = append(parts, cell+strings.Repeat(" ", w-len(cell)))
		}
	}
	line := strings.TrimRight(strings.Join(parts, "  "), " ")
	b.WriteString(line)
	b.WriteString("\n")
}

// sortStrings sorts a string slice in place. Kept local to avoid pulling sort
// into callers and to make the column ordering intent obvious.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

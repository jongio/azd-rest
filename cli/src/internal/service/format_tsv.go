package service

// renderTSV renders a JSON response body as tab-separated values.
//
// It accepts a top-level JSON array, an object that wraps rows under a common
// key (value, data, results, or items), or a single object (rendered as one
// row). Rows that are objects become columns following the same ordering as the
// table and CSV formats; rows that are scalars render under a single "value"
// column. Nested objects and arrays are written as compact JSON inside the cell.
// An empty result set produces no output.
func renderTSV(body []byte) (string, error) {
	return renderDelimited(body, "tsv", '\t')
}

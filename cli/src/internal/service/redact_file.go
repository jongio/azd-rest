package service

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// loadRedactFile reads response redaction paths from a file, one dotted path per
// line. Blank lines and lines beginning with "#" are ignored.
func loadRedactFile(path string) ([]string, error) {
	file, err := os.Open(path) // #nosec G304 -- User-specified file path via --redact-file flag is intentional.
	if err != nil {
		return nil, fmt.Errorf("failed to open redact file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var result []string
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.ContainsAny(line, " \t") {
			return nil, fmt.Errorf("invalid redact path on line %d of %s: %q (expected one dotted path)", lineNum, path, line)
		}
		result = append(result, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read redact file: %w", err)
	}
	return result, nil
}

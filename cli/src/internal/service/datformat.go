package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jongio/azd-rest/src/internal/config"
)

// Supported values for the --data-format flag (#236).
const (
	dataFormatJSON = "json"
	dataFormatYAML = "yaml"
)

// dataFormatError signals invalid --data-format usage: an unknown format value,
// a conflict with the field-body flags, or a request body that will not parse as
// YAML. It reports exit code 2 through the ExitCoder contract so main can map it
// to a usage failure.
type dataFormatError struct{ err error }

// Error returns the underlying message.
func (e *dataFormatError) Error() string { return e.err.Error() }

// Unwrap exposes the wrapped error for errors.Is/As.
func (e *dataFormatError) Unwrap() error { return e.err }

// ExitCode returns 2 to match the CLI's convention for invalid usage.
func (e *dataFormatError) ExitCode() int { return 2 }

// yamlToJSON converts a YAML document to its equivalent JSON encoding. YAML is a
// superset of JSON, so a body that is already JSON round-trips unchanged.
func yamlToJSON(raw []byte) ([]byte, error) {
	var v any
	if err := yaml.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// readRequestBody returns the raw request body bytes sourced from --data-file
// (with @{file} shorthand support) or the inline --data value. It returns a nil
// slice when neither is set.
func readRequestBody(cfg config.Config) ([]byte, error) {
	if cfg.DataFile != "" {
		filePath := strings.TrimPrefix(cfg.DataFile, "@")
		raw, err := os.ReadFile(filePath) // #nosec G304 -- User-specified file path via --data-file flag is intentional.
		if err != nil {
			return nil, fmt.Errorf("failed to read data file: %w", err)
		}
		return raw, nil
	}
	if cfg.Data != "" {
		return []byte(cfg.Data), nil
	}
	return nil, nil
}

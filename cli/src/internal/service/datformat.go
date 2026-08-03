package service

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jongio/azd-rest/src/internal/config"
)

var stdinReader io.Reader = os.Stdin

// Supported values for the --data-format flag (#236).
const (
	dataFormatJSON = "json"
	dataFormatYAML = "yaml"
)

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
		if readsFromStdin(cfg.DataFile) {
			if cfg.Data != "" {
				return nil, fmt.Errorf("--data-file - cannot be combined with --data")
			}
			return readStdinBody()
		}
		filePath := strings.TrimPrefix(cfg.DataFile, "@")
		raw, err := os.ReadFile(filePath) // #nosec G304 -- User-specified file path via --data-file flag is intentional.
		if err != nil {
			return nil, fmt.Errorf("failed to read data file: %w", err)
		}
		return raw, nil
	}
	if cfg.Data != "" {
		if cfg.Data == "@-" {
			return readStdinBody()
		}
		return []byte(cfg.Data), nil
	}
	return nil, nil
}

func readsFromStdin(path string) bool {
	return path == "-" || path == "@-"
}

func readStdinBody() ([]byte, error) {
	raw, err := io.ReadAll(stdinReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body from stdin: %w", err)
	}
	return raw, nil
}

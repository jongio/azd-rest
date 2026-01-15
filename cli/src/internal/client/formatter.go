package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	FormatAuto OutputFormat = "auto"
	FormatJSON OutputFormat = "json"
	FormatRaw  OutputFormat = "raw"
)

// Formatter handles response formatting and output
type Formatter struct {
	verbose bool
	format  OutputFormat
}

// NewFormatter creates a new formatter
func NewFormatter(verbose bool, format string) *Formatter {
	outputFormat := FormatAuto
	if format != "" {
		outputFormat = OutputFormat(format)
	}

	return &Formatter{
		verbose: verbose,
		format:  outputFormat,
	}
}

// Format formats the response for output
func (f *Formatter) Format(resp *Response) (string, error) {
	var output strings.Builder

	// Verbose mode - show headers and timing
	if f.verbose {
		output.WriteString(fmt.Sprintf("Status: %s\n", resp.Status))
		output.WriteString(fmt.Sprintf("Duration: %v\n", resp.Duration))
		output.WriteString("\nResponse Headers:\n")
		for key, values := range resp.Headers {
			for _, value := range values {
				// Redact Authorization header
				if strings.EqualFold(key, "authorization") && len(value) > 8 {
					value = value[:4] + "..." + value[len(value)-4:]
				}
				output.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
			}
		}
		output.WriteString("\n")
	}

	// Format body
	body := resp.Body
	contentType := resp.Headers.Get("Content-Type")

	// Determine format
	format := f.format
	if format == FormatAuto {
		if strings.Contains(contentType, "application/json") {
			format = FormatJSON
		} else {
			format = FormatRaw
		}
	}

	// Format based on type
	switch format {
	case FormatJSON:
		formatted, err := f.formatJSON(body)
		if err != nil {
			// If JSON formatting fails, fall back to raw
			output.Write(body)
		} else {
			output.WriteString(formatted)
		}
	case FormatRaw:
		output.Write(body)
	default:
		output.Write(body)
	}

	return output.String(), nil
}

// formatJSON pretty-prints JSON
func (f *Formatter) formatJSON(data []byte) (string, error) {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", err
	}

	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

// WriteOutput writes the formatted output to the appropriate destination
func (f *Formatter) WriteOutput(output string, outputFile string) error {
	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(output), 0644)
	}

	fmt.Print(output)
	return nil
}

// WriteRawOutput writes raw bytes to a file or stdout
func (f *Formatter) WriteRawOutput(data []byte, outputFile string) error {
	if outputFile != "" {
		return os.WriteFile(outputFile, data, 0644)
	}

	_, err := io.Copy(os.Stdout, bytes.NewReader(data))
	return err
}

// RedactToken redacts sensitive parts of an authorization token
func RedactToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// IsJSON checks if content appears to be JSON
func IsJSON(data []byte) bool {
	var js interface{}
	return json.Unmarshal(data, &js) == nil
}

package client

import "github.com/jongio/azd-core/httpclient"

// OutputFormat specifies the desired output format (auto, json, or raw).
type OutputFormat = httpclient.OutputFormat

// Formatter formats HTTP responses for display output.
type Formatter = httpclient.Formatter

// Output format constants.
const (
	FormatAuto = httpclient.FormatAuto
	FormatJSON = httpclient.FormatJSON
	FormatRaw  = httpclient.FormatRaw
)

// NewFormatter creates a Formatter for the given OutputFormat.
var NewFormatter = httpclient.NewFormatter

// RedactSensitiveHeader replaces sensitive HTTP header values with redacted placeholders.
var RedactSensitiveHeader = httpclient.RedactSensitiveHeader

// RedactToken replaces bearer token values with redacted placeholders.
var RedactToken = httpclient.RedactToken

// IsJSON reports whether the given content type indicates JSON data.
var IsJSON = httpclient.IsJSON

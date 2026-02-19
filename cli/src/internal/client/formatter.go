package client

import "github.com/jongio/azd-core/httpclient"

// Re-export formatter types from httpclient
type OutputFormat = httpclient.OutputFormat
type Formatter = httpclient.Formatter

// Re-export constants
const (
	FormatAuto = httpclient.FormatAuto
	FormatJSON = httpclient.FormatJSON
	FormatRaw  = httpclient.FormatRaw
)

// Re-export functions
var NewFormatter = httpclient.NewFormatter
var RedactSensitiveHeader = httpclient.RedactSensitiveHeader
var RedactToken = httpclient.RedactToken
var IsJSON = httpclient.IsJSON

// Package config defines the explicit configuration struct for the azd rest CLI,
// replacing global mutable flag variables with a value type that is populated
// once at startup and threaded through the call graph via dependency injection.
package config

import "time"

// Config holds all CLI flag values as an explicit, immutable-after-init struct.
// It is populated from cobra persistent flags in the root command and passed
// to the service layer - no global mutable state is involved.
type Config struct {
	Scope           string
	NoAuth          bool
	APIVersion      string
	BaseURL         string
	ClientRequestID string
	Traceparent     string
	URLParams       []string
	URLParamFile    string
	Headers         []string
	Accept          string
	ContentType     string
	HeaderFile      string
	HeaderEnv       []string
	Data            string
	DataFile        string
	DataFormat      string
	Query           string
	FormFields      []string
	JSONFields      []string
	JSONFieldsRaw   []string
	OutputFile      string
	OutputFormat    string
	Verbose         bool
	Flatten         bool
	Paginate        bool
	Retry           int
	Binary          bool
	Insecure        bool
	Silent          bool
	Timeout         time.Duration
	MaxTime         time.Duration
	MaxLatency      string
	FollowRedirects bool
	MaxRedirects    int
	MaxPages        int
	MaxResponseSize int64
	ReadOnly        bool
	ShowThrottle    bool
	Repeat          int
	RepeatDelay     time.Duration
	Color           string
	WriteOut        string
	Include         bool
	AllowedHosts    []string
	Redact          []string
	RedactSecrets   bool
	Omit            []string
	RedactFile      string
	Fields          []string
	TableColumns    []string
	DumpHeaders     string
	ExpectedHeaders []string
	MetadataFile    string
	Fail            bool
	Diff            string
	AllowStatus     string
	ValidateSchema  string
	DryRun          bool
	Expect          []string
	RawOutput       bool
	Compact         bool
	Limit           int
	ShowRequestIDs  bool
	NoBody          bool
}

// Defaults returns a Config populated with the default flag values.
func Defaults() Config {
	return Config{
		OutputFormat:    "auto",
		Retry:           3,
		Timeout:         30 * time.Second,
		FollowRedirects: true,
		MaxRedirects:    10,
		MaxPages:        100,
		MaxResponseSize: 100 * 1024 * 1024, // 100MB
		Repeat:          1,
		Color:           "auto",
	}
}

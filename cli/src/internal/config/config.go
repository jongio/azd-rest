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
	URLParams       []string
	Headers         []string
	Data            string
	DataFile        string
	Query           string
	FormFields      []string
	OutputFile      string
	OutputFormat    string
	Verbose         bool
	Paginate        bool
	Retry           int
	Binary          bool
	Insecure        bool
	Silent          bool
	Timeout         time.Duration
	MaxTime         time.Duration
	FollowRedirects bool
	MaxRedirects    int
	MaxPages        int
	MaxResponseSize int64
	ShowThrottle    bool
	Repeat          int
	Color           string
	WriteOut        string
	Include         bool
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

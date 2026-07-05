package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
)

// ANSI color codes used to highlight JSON output.
const (
	colorReset  = "\x1b[0m"
	colorKey    = "\x1b[36m" // cyan
	colorString = "\x1b[32m" // green
	colorNumber = "\x1b[33m" // yellow
	colorLit    = "\x1b[35m" // magenta (true/false/null)
)

// Recognized values for the --color flag.
const (
	colorModeAuto   = "auto"
	colorModeAlways = "always"
	colorModeNever  = "never"
)

// validateColorMode reports an error for an unrecognized --color value.
// An empty value is treated as the default (auto).
func validateColorMode(mode string) error {
	switch mode {
	case "", colorModeAuto, colorModeAlways, colorModeNever:
		return nil
	default:
		return fmt.Errorf("invalid --color value %q (expected auto, always, or never)", mode)
	}
}

// shouldColorize decides whether the formatted JSON output should be colorized.
// The mode is assumed valid (validateColorMode runs earlier). Color is never
// applied to a file, to non-JSON content, or to raw-format output.
func shouldColorize(cfg config.Config, resp *client.Response) bool {
	mode := cfg.Color
	if mode == "" {
		mode = colorModeAuto
	}
	if mode == colorModeNever {
		return false
	}
	if cfg.OutputFile != "" {
		return false
	}
	if cfg.OutputFormat == string(client.FormatRaw) || !client.IsJSON(resp.Body) {
		return false
	}
	if mode == colorModeAlways {
		return true
	}
	// auto
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return stdoutIsTerminal()
}

// stdoutIsTerminal reports whether stdout is a character device (a TTY).
func stdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// isNumberByte reports whether b can appear in a JSON number token.
func isNumberByte(b byte) bool {
	return (b >= '0' && b <= '9') || b == '-' || b == '+' || b == '.' || b == 'e' || b == 'E'
}

// colorizeJSON wraps JSON tokens in ANSI color codes while preserving the exact
// structure and whitespace of the input. Object keys, string values, numbers,
// and the literals true/false/null are colored; punctuation is left untouched.
func colorizeJSON(s string) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/4)
	i, n := 0, len(s)
	for i < n {
		c := s[i]
		switch {
		case c == '"':
			// Consume the full string literal, honoring escapes.
			j := i + 1
			for j < n {
				if s[j] == '\\' {
					j += 2
					continue
				}
				if s[j] == '"' {
					j++
					break
				}
				j++
			}
			literal := s[i:j]
			// A string is a key when the next non-space byte is a colon.
			k := j
			for k < n && (s[k] == ' ' || s[k] == '\t' || s[k] == '\n' || s[k] == '\r') {
				k++
			}
			if k < n && s[k] == ':' {
				b.WriteString(colorKey + literal + colorReset)
			} else {
				b.WriteString(colorString + literal + colorReset)
			}
			i = j
		case (c >= '0' && c <= '9') || c == '-':
			j := i
			for j < n && isNumberByte(s[j]) {
				j++
			}
			b.WriteString(colorNumber + s[i:j] + colorReset)
			i = j
		case c == 't' && strings.HasPrefix(s[i:], "true"):
			b.WriteString(colorLit + "true" + colorReset)
			i += 4
		case c == 'f' && strings.HasPrefix(s[i:], "false"):
			b.WriteString(colorLit + "false" + colorReset)
			i += 5
		case c == 'n' && strings.HasPrefix(s[i:], "null"):
			b.WriteString(colorLit + "null" + colorReset)
			i += 4
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

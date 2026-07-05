package service

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/jongio/azd-rest/src/internal/client"
)

var writeOutTokenRE = regexp.MustCompile(`%\{([^}]+)\}`)

// ExpandWriteOut expands a curl-style --write-out template using response
// metadata. Supported variables include http_code, http_status, time_total,
// time_total_ms, size_download, content_type, method, url, and header.NAME.
// Unknown %{...} tokens are left unchanged. The escape sequences \n and \t are
// expanded to newline and tab.
func ExpandWriteOut(format, method, url string, resp *client.Response) string {
	expanded := writeOutTokenRE.ReplaceAllStringFunc(format, func(token string) string {
		name := token[2 : len(token)-1] // strip leading %{ and trailing }
		if val, ok := writeOutValue(name, method, url, resp); ok {
			return val
		}
		return token // unknown token is left literal
	})
	expanded = strings.ReplaceAll(expanded, `\n`, "\n")
	expanded = strings.ReplaceAll(expanded, `\t`, "\t")
	return expanded
}

// writeOutValue resolves a single --write-out variable name. The bool result
// reports whether the name is a recognized variable; recognized variables that
// have no value (such as an absent header) resolve to an empty string.
func writeOutValue(name, method, url string, resp *client.Response) (string, bool) {
	if headerName, ok := strings.CutPrefix(name, "header."); ok {
		value := resp.Headers.Get(headerName)
		if value == "" {
			return "", true
		}
		return client.RedactSensitiveHeader(headerName, value), true
	}

	switch name {
	case "http_code":
		return strconv.Itoa(resp.StatusCode), true
	case "http_status":
		return resp.Status, true
	case "time_total":
		return strconv.FormatFloat(resp.Duration.Seconds(), 'f', 6, 64), true
	case "time_total_ms":
		return strconv.FormatInt(resp.Duration.Milliseconds(), 10), true
	case "size_download":
		return strconv.Itoa(len(resp.Body)), true
	case "content_type":
		return resp.Headers.Get("Content-Type"), true
	case "method":
		return method, true
	case "url":
		return url, true
	default:
		return "", false
	}
}

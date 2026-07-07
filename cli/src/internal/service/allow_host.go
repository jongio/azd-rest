package service

import (
	"net/url"
	"strings"
)

// requestHostAllowed parses rawURL and reports whether its host matches the
// allowlist. The extracted host is returned for use in diagnostics. The port,
// if any, is ignored. This lives here rather than inline so it can use the
// net/url package, which the request builder shadows with a url parameter.
func requestHostAllowed(rawURL string, patterns []string) (host string, allowed bool, err error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false, err
	}
	host = parsed.Hostname()
	return host, hostMatchesAllowlist(host, patterns), nil
}

// hostMatchesAllowlist reports whether host matches any pattern in the
// allowlist. Matching is case insensitive. A pattern that begins with "*."
// matches any subdomain of the remaining suffix, for example "*.vault.azure.net"
// matches "kv.vault.azure.net" but not the bare "vault.azure.net". Any other
// pattern must match the host exactly. Blank patterns are ignored.
func hostMatchesAllowlist(host string, patterns []string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	for _, pattern := range patterns {
		p := strings.ToLower(strings.TrimSpace(pattern))
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "*.") {
			suffix := p[1:]
			if len(host) > len(suffix) && strings.HasSuffix(host, suffix) {
				return true
			}
			continue
		}
		if host == p {
			return true
		}
	}
	return false
}

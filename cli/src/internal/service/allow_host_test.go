package service

import "testing"

func TestHostMatchesAllowlist(t *testing.T) {
	cases := []struct {
		name     string
		host     string
		patterns []string
		want     bool
	}{
		{"exact match", "management.azure.com", []string{"management.azure.com"}, true},
		{"exact mismatch", "evil.com", []string{"management.azure.com"}, false},
		{"case insensitive host", "Management.Azure.Com", []string{"management.azure.com"}, true},
		{"case insensitive pattern", "management.azure.com", []string{"MANAGEMENT.AZURE.COM"}, true},
		{"wildcard subdomain", "kv.vault.azure.net", []string{"*.vault.azure.net"}, true},
		{"wildcard deep subdomain", "a.b.vault.azure.net", []string{"*.vault.azure.net"}, true},
		{"wildcard does not match apex", "vault.azure.net", []string{"*.vault.azure.net"}, false},
		{"wildcard mismatch suffix", "kv.vault.example.net", []string{"*.vault.azure.net"}, false},
		{"multiple patterns first", "graph.microsoft.com", []string{"graph.microsoft.com", "*.azure.com"}, true},
		{"multiple patterns second", "kv.azure.com", []string{"graph.microsoft.com", "*.azure.com"}, true},
		{"blank patterns ignored", "management.azure.com", []string{"", "  "}, false},
		{"blank pattern skipped then match", "management.azure.com", []string{"", "management.azure.com"}, true},
		{"empty host", "", []string{"management.azure.com"}, false},
		{"whitespace pattern trimmed", "management.azure.com", []string{"  management.azure.com  "}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hostMatchesAllowlist(tc.host, tc.patterns); got != tc.want {
				t.Fatalf("hostMatchesAllowlist(%q, %v) = %v, want %v", tc.host, tc.patterns, got, tc.want)
			}
		})
	}
}

func TestRequestHostAllowed(t *testing.T) {
	cases := []struct {
		name     string
		rawURL   string
		patterns []string
		wantHost string
		wantOK   bool
		wantErr  bool
	}{
		{"host extracted and allowed", "https://management.azure.com/subscriptions", []string{"management.azure.com"}, "management.azure.com", true, false},
		{"port ignored", "https://management.azure.com:8443/x", []string{"management.azure.com"}, "management.azure.com", true, false},
		{"disallowed host", "https://evil.com/x", []string{"management.azure.com"}, "evil.com", false, false},
		{"wildcard match", "https://kv.vault.azure.net/secrets", []string{"*.vault.azure.net"}, "kv.vault.azure.net", true, false},
		{"invalid url", "://no-scheme", []string{"management.azure.com"}, "", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host, ok, err := requestHostAllowed(tc.rawURL, tc.patterns)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.rawURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if host != tc.wantHost {
				t.Fatalf("host = %q, want %q", host, tc.wantHost)
			}
			if ok != tc.wantOK {
				t.Fatalf("allowed = %v, want %v", ok, tc.wantOK)
			}
		})
	}
}

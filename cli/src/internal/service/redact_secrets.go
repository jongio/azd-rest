package service

// cspell:ignore connectionstring sharedaccesskey sharedaccesssignature accountkey primarykey secondarykey accesskey apikey encryptionkey accesstoken refreshtoken idtoken bearertoken sastoken

import (
	"bytes"
	"encoding/json"
	"strings"
)

// sensitiveKeyTerms are normalized substrings that mark a JSON object key as
// likely to hold a secret value. Matching runs against a normalized key
// (lowercased with non-alphanumeric characters removed) so connectionString,
// connection_string, and connection-string all match "connectionstring". The
// bare words "key" and "token" are deliberately excluded to avoid masking
// common identifier fields such as "partitionKey" or a CSRF "token" name.
var sensitiveKeyTerms = []string{
	"password", "passwd", "pwd",
	"secret",
	"credential",
	"connectionstring",
	"sharedaccesskey", "sharedaccesssignature",
	"accountkey", "primarykey", "secondarykey", "accesskey", "apikey",
	"privatekey", "encryptionkey",
	"accesstoken", "refreshtoken", "idtoken", "bearertoken", "sastoken",
}

// redactSecretsJSONBody parses body as JSON and replaces the value of every
// object key whose name looks sensitive with a fixed placeholder, at any depth.
// It returns re-encoded JSON. An error is returned only when body is not valid
// JSON, so callers can leave the body unchanged.
func redactSecretsJSONBody(body []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return nil, err
	}

	redactSecretsNode(parsed)

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(parsed); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// redactSecretsNode walks node in place. For objects it masks the value of any
// sensitive key and recurses into the rest; for arrays it recurses into every
// element. Masking a key replaces its whole value, so a nested object or array
// under a sensitive key is masked as a unit.
func redactSecretsNode(node any) {
	switch v := node.(type) {
	case map[string]any:
		for key, child := range v {
			if isSensitiveKey(key) {
				v[key] = redactedPlaceholder
				continue
			}
			redactSecretsNode(child)
		}
	case []any:
		for _, item := range v {
			redactSecretsNode(item)
		}
	}
}

// isSensitiveKey reports whether a JSON key name contains a known sensitive
// term after normalization.
func isSensitiveKey(key string) bool {
	normalized := normalizeKey(key)
	if normalized == "" {
		return false
	}
	for _, term := range sensitiveKeyTerms {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	return false
}

// normalizeKey lowercases key and drops every character that is not a lowercase
// letter or digit, so different casing and separators compare equal.
func normalizeKey(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for _, r := range strings.ToLower(key) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

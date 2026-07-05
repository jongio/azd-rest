package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
)

// makeJWT builds a syntactically valid (unsigned) JWT with the given claims so
// tests can exercise decoding without acquiring a real Azure token.
func makeJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	seg := base64.RawURLEncoding.EncodeToString(payload)
	return "aGVhZGVy." + seg + ".c2ln"
}

func TestDecodeJWTClaims_Valid(t *testing.T) {
	token := makeJWT(t, map[string]any{"tid": "tenant-123", "oid": "obj-456"})
	claims, err := decodeJWTClaims(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims["tid"] != "tenant-123" || claims["oid"] != "obj-456" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestDecodeJWTClaims_Errors(t *testing.T) {
	cases := map[string]string{
		"empty":       "",
		"two-segment": "header.payload",
		"bad-base64":  "header.@@@@.sig",
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeJWTClaims(token); err == nil {
				t.Fatalf("expected error for %q", name)
			}
		})
	}
}

func TestClaimsToIdentity_User(t *testing.T) {
	exp := time.Now().Add(time.Hour).Unix()
	id := claimsToIdentity(map[string]any{
		"tid":                "t1",
		"oid":                "o1",
		"appid":              "a1",
		"aud":                "https://management.azure.com",
		"preferred_username": "user@contoso.com",
		"scp":                "user_impersonation offline_access",
		"exp":                float64(exp),
	})
	if id.Tenant != "t1" || id.ObjectID != "o1" || id.AppID != "a1" {
		t.Fatalf("unexpected identity: %#v", id)
	}
	if id.Username != "user@contoso.com" {
		t.Fatalf("unexpected username: %q", id.Username)
	}
	if len(id.Scopes) != 2 || id.Scopes[0] != "user_impersonation" {
		t.Fatalf("unexpected scopes: %#v", id.Scopes)
	}
	if id.ExpiresAt.Unix() != exp {
		t.Fatalf("unexpected expiry: %v", id.ExpiresAt)
	}
}

func TestClaimsToIdentity_AppRolesAndAudArray(t *testing.T) {
	id := claimsToIdentity(map[string]any{
		"azp":   "app-1",
		"aud":   []any{"aud-first", "aud-second"},
		"roles": []any{"Reader", "Contributor"},
	})
	if id.AppID != "app-1" {
		t.Fatalf("expected azp fallback for appId, got %q", id.AppID)
	}
	if id.Audience != "aud-first" {
		t.Fatalf("expected first audience, got %q", id.Audience)
	}
	if len(id.Roles) != 2 || id.Roles[1] != "Contributor" {
		t.Fatalf("unexpected roles: %#v", id.Roles)
	}
	if len(id.Scopes) != 0 {
		t.Fatalf("expected no delegated scopes, got %#v", id.Scopes)
	}
}

func TestRunWhoami_Text(t *testing.T) {
	token := makeJWT(t, map[string]any{
		"tid":   "tenant-abc",
		"oid":   "object-xyz",
		"appid": "app-def",
		"aud":   "https://management.azure.com",
	})
	tp := &client.MockTokenProvider{Token: token}
	var buf bytes.Buffer
	if err := runWhoami(context.Background(), tp, managementScope, "", &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Tenant:", "tenant-abc", "Object ID:", "object-xyz", "App ID:", "app-def"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, token) {
		t.Fatalf("raw token must never be printed")
	}
}

func TestRunWhoami_JSON(t *testing.T) {
	token := makeJWT(t, map[string]any{"tid": "tenant-abc", "oid": "object-xyz"})
	tp := &client.MockTokenProvider{Token: token}
	var buf bytes.Buffer
	if err := runWhoami(context.Background(), tp, managementScope, "json", &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got identity
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got.Tenant != "tenant-abc" || got.ObjectID != "object-xyz" {
		t.Fatalf("unexpected decoded identity: %#v", got)
	}
	if strings.Contains(buf.String(), token) {
		t.Fatalf("raw token must never be printed")
	}
}

func TestRunWhoami_TokenError(t *testing.T) {
	tp := &client.MockTokenProvider{Error: errors.New("not logged in")}
	var buf bytes.Buffer
	if err := runWhoami(context.Background(), tp, managementScope, "", &buf); err == nil {
		t.Fatal("expected error when token acquisition fails")
	}
}

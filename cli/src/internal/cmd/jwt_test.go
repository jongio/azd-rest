package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunJWT_Text(t *testing.T) {
	token := makeJWT(t, map[string]any{
		"tid":   "tenant-abc",
		"oid":   "object-xyz",
		"appid": "app-def",
		"aud":   "https://management.azure.com",
	})
	var buf bytes.Buffer
	if err := runJWT(token, "", &buf); err != nil {
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

func TestRunJWT_JSONClaims(t *testing.T) {
	token := makeJWT(t, map[string]any{
		"tid": "tenant-abc",
		"oid": "object-xyz",
		"scp": "user_impersonation",
	})
	var buf bytes.Buffer
	if err := runJWT(token, "json", &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got["tid"] != "tenant-abc" || got["oid"] != "object-xyz" || got["scp"] != "user_impersonation" {
		t.Fatalf("unexpected decoded claims: %#v", got)
	}
	if strings.Contains(buf.String(), token) {
		t.Fatalf("raw token must never be printed")
	}
}

func TestRunJWT_MalformedToken(t *testing.T) {
	cases := map[string]string{
		"empty":       "",
		"two-segment": "header.payload",
		"bad-base64":  "header.@@@@.sig",
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := runJWT(token, "", &buf); err == nil {
				t.Fatalf("expected error for %q", name)
			}
		})
	}
}

func TestNewJWTCommand_RequiresToken(t *testing.T) {
	cmd := NewJWTCommand()
	if cmd.Use != "jwt <token>" {
		t.Fatalf("unexpected Use: %q", cmd.Use)
	}
	if err := cmd.Args(cmd, nil); err == nil {
		t.Fatal("expected error when no token argument is given")
	}
	if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
		t.Fatal("expected error when more than one argument is given")
	}
}

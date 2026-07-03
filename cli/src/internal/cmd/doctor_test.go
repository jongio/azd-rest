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

// makeDoctorJWT builds a minimal unsigned JWT with the given claims for testing.
func makeDoctorJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return header + "." + payload + ".sig"
}

func TestRunDoctorAllHealthy(t *testing.T) {
	token := makeDoctorJWT(t, map[string]any{
		"tid": "tenant-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tp := &client.MockTokenProvider{Token: token}

	var buf bytes.Buffer
	err := runDoctor(context.Background(), tp, nil, "text", &buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Azure authentication") {
		t.Errorf("missing authentication check:\n%s", out)
	}
	if !strings.Contains(out, "tenant-123") {
		t.Errorf("expected tenant in output:\n%s", out)
	}
	if strings.Contains(out, "[fail]") {
		t.Errorf("did not expect failures:\n%s", out)
	}
}

func TestRunDoctorAuthFailureReturnsError(t *testing.T) {
	tp := &client.MockTokenProvider{Error: errors.New("no credentials found")}

	var buf bytes.Buffer
	err := runDoctor(context.Background(), tp, nil, "text", &buf)
	if err == nil {
		t.Fatalf("expected an error when authentication fails")
	}
	out := buf.String()
	if !strings.Contains(out, "[fail]") {
		t.Errorf("expected a failed check in output:\n%s", out)
	}
	if !strings.Contains(out, "az login") {
		t.Errorf("expected remediation guidance:\n%s", out)
	}
	// A failed token acquisition should not produce a token-claims check.
	if strings.Contains(out, "Token claims") {
		t.Errorf("should not report token claims when auth failed:\n%s", out)
	}
}

func TestRunDoctorFactoryErrorReturnsError(t *testing.T) {
	var buf bytes.Buffer
	err := runDoctor(context.Background(), nil, errors.New("factory boom"), "text", &buf)
	if err == nil {
		t.Fatalf("expected an error when the token provider factory fails")
	}
	if !strings.Contains(buf.String(), "factory boom") {
		t.Errorf("expected factory error detail:\n%s", buf.String())
	}
}

func TestRunDoctorExpiredTokenWarns(t *testing.T) {
	token := makeDoctorJWT(t, map[string]any{
		"tid": "tenant-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	tp := &client.MockTokenProvider{Token: token}

	var buf bytes.Buffer
	err := runDoctor(context.Background(), tp, nil, "text", &buf)
	if err != nil {
		t.Fatalf("an expired token is a warning, not a failure, got: %v", err)
	}
	if !strings.Contains(buf.String(), "[warn]") || !strings.Contains(buf.String(), "expired") {
		t.Errorf("expected an expired-token warning:\n%s", buf.String())
	}
}

func TestRunDoctorJSONOutput(t *testing.T) {
	token := makeDoctorJWT(t, map[string]any{"tid": "t", "exp": time.Now().Add(time.Hour).Unix()})
	tp := &client.MockTokenProvider{Token: token}

	var buf bytes.Buffer
	if err := runDoctor(context.Background(), tp, nil, "json", &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var checks []doctorCheck
	if err := json.Unmarshal(buf.Bytes(), &checks); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(checks) < 2 {
		t.Fatalf("expected at least 2 checks, got %d", len(checks))
	}
	for _, c := range checks {
		if c.Name == "" || c.Status == "" {
			t.Errorf("check missing name or status: %+v", c)
		}
	}
}

func TestDecodeTokenClaimsInvalid(t *testing.T) {
	if _, ok := decodeTokenClaims("not-a-jwt"); ok {
		t.Errorf("expected decode to fail for a non-JWT string")
	}
	if _, ok := decodeTokenClaims("header.@@@@.sig"); ok {
		t.Errorf("expected decode to fail for invalid base64 payload")
	}
}

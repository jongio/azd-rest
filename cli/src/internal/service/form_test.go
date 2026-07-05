package service

import (
	"io"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
)

func TestEncodeFormFields(t *testing.T) {
	tests := []struct {
		name    string
		fields  []string
		want    string
		wantErr bool
	}{
		{"single", []string{"a=1"}, "a=1", false},
		{"multiple sorted", []string{"b=2", "a=1"}, "a=1&b=2", false},
		{"space encoded", []string{"b=hello world"}, "b=hello+world", false},
		{"multi value", []string{"a=1", "a=2"}, "a=1&a=2", false},
		{"special chars", []string{"q=a&b=c"}, "q=a%26b%3Dc", false},
		{"empty value", []string{"a="}, "a=", false},
		{"missing equals", []string{"abc"}, "", true},
		{"empty key", []string{"=v"}, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := encodeFormFields(tc.fields)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %v", tc.fields)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("encodeFormFields(%v) = %q, want %q", tc.fields, got, tc.want)
			}
		})
	}
}

func TestHasHeader(t *testing.T) {
	headers := map[string]string{"content-type": "text/plain"}
	if !hasHeader(headers, "Content-Type") {
		t.Error("expected case-insensitive match for Content-Type")
	}
	if hasHeader(headers, "Accept") {
		t.Error("did not expect a match for Accept")
	}
}

func TestBuildRequestOptions_FormFieldsBody(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		FormFields: []string{"grant_type=client_credentials", "scope=api world"},
	}

	opts, cleanup, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if opts.Headers[contentTypeHeader] != formURLEncoded {
		t.Errorf("Content-Type = %q, want %q", opts.Headers[contentTypeHeader], formURLEncoded)
	}

	body, err := io.ReadAll(opts.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	want := "grant_type=client_credentials&scope=api+world"
	if string(body) != want {
		t.Errorf("body = %q, want %q", string(body), want)
	}
}

func TestBuildRequestOptions_FormFieldsKeepsUserContentType(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		Headers:    []string{"Content-Type: application/json"},
		FormFields: []string{"a=1"},
	}

	opts, cleanup, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if opts.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", opts.Headers["Content-Type"])
	}
}

func TestBuildRequestOptions_FormFieldsConflict(t *testing.T) {
	svc := newTestService()

	conflicts := []config.Config{
		{NoAuth: true, FormFields: []string{"a=1"}, Data: `{"x":1}`},
		{NoAuth: true, FormFields: []string{"a=1"}, DataFile: "body.json"},
	}
	for _, cfg := range conflicts {
		_, _, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x")
		if err == nil {
			t.Errorf("expected conflict error for cfg %+v", cfg)
		}
	}
}

func TestBuildRequestOptions_FormFieldsInvalid(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		FormFields: []string{"no-equals-sign"},
	}
	if _, _, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x"); err == nil {
		t.Error("expected error for invalid form field")
	}
}

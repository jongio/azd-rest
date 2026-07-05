package service

import (
	"io"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
)

func TestEncodeJSONFields(t *testing.T) {
	tests := []struct {
		name    string
		fields  []string
		want    string
		wantErr bool
	}{
		{"single string", []string{"name=web"}, `{"name":"web"}`, false},
		{"raw number", []string{"count:=3"}, `{"count":3}`, false},
		{"raw bool", []string{"enabled:=true"}, `{"enabled":true}`, false},
		{"raw null", []string{"note:=null"}, `{"note":null}`, false},
		{"raw array", []string{`tags:=["a","b"]`}, `{"tags":["a","b"]}`, false},
		{"raw object", []string{`sku:={"name":"S1"}`}, `{"sku":{"name":"S1"}}`, false},
		{"mixed sorted", []string{"count:=3", "name=web"}, `{"count":3,"name":"web"}`, false},
		{"string with equals", []string{"expr=a=b"}, `{"expr":"a=b"}`, false},
		{"empty string value", []string{"name="}, `{"name":""}`, false},
		{"last key wins", []string{"a=1", "a=2"}, `{"a":"2"}`, false},
		{"value that looks numeric stays string", []string{"zip=01234"}, `{"zip":"01234"}`, false},
		{"missing equals", []string{"token"}, "", true},
		{"empty key", []string{"=v"}, "", true},
		{"empty raw key", []string{":=1"}, "", true},
		{"invalid raw json", []string{"bad:={oops}"}, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := encodeJSONFields(tc.fields)
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
				t.Errorf("encodeJSONFields(%v) = %q, want %q", tc.fields, got, tc.want)
			}
		})
	}
}

func TestBuildRequestOptions_JSONFieldsBody(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		JSONFields: []string{"name=web", "count:=3"},
	}

	opts, cleanup, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if opts.Headers[contentTypeHeader] != applicationJSON {
		t.Errorf("Content-Type = %q, want %q", opts.Headers[contentTypeHeader], applicationJSON)
	}

	body, err := io.ReadAll(opts.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	want := `{"count":3,"name":"web"}`
	if string(body) != want {
		t.Errorf("body = %q, want %q", string(body), want)
	}
}

func TestBuildRequestOptions_JSONFieldsKeepsUserContentType(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		Headers:    []string{"Content-Type: application/merge-patch+json"},
		JSONFields: []string{"a=1"},
	}

	opts, cleanup, err := svc.BuildRequestOptions(cfg, "PATCH", "https://example.com/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if opts.Headers["Content-Type"] != "application/merge-patch+json" {
		t.Errorf("Content-Type = %q, want application/merge-patch+json", opts.Headers["Content-Type"])
	}
}

func TestBuildRequestOptions_JSONFieldsConflict(t *testing.T) {
	svc := newTestService()

	conflicts := []config.Config{
		{NoAuth: true, JSONFields: []string{"a=1"}, Data: `{"x":1}`},
		{NoAuth: true, JSONFields: []string{"a=1"}, DataFile: "body.json"},
		{NoAuth: true, JSONFields: []string{"a=1"}, FormFields: []string{"b=2"}},
	}
	for _, cfg := range conflicts {
		if _, _, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x"); err == nil {
			t.Errorf("expected conflict error for cfg %+v", cfg)
		}
	}
}

func TestBuildRequestOptions_JSONFieldsInvalid(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:     true,
		JSONFields: []string{"bad:={oops}"},
	}
	if _, _, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x"); err == nil {
		t.Error("expected error for invalid json field")
	}
}

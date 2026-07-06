package service

import (
	"io"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
)

func TestBuildJSONBodyStringFields(t *testing.T) {
	got, err := buildJSONBody([]string{"name=alpha", "region=eastus"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"name":"alpha","region":"eastus"}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestBuildJSONBodyRawFieldsKeepType(t *testing.T) {
	got, err := buildJSONBody(nil, []string{
		"count:=3",
		"enabled:=true",
		"tags:=[\"a\",\"b\"]",
		"meta:={\"k\":1}",
		"empty:=null",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"count":3,"empty":null,"enabled":true,"meta":{"k":1},"tags":["a","b"]}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestBuildJSONBodyNestingAndMerge(t *testing.T) {
	got, err := buildJSONBody(
		[]string{"sku.name=Standard_LRS"},
		[]string{"sku.tier:=1", "properties.enabled:=true"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"properties":{"enabled":true},"sku":{"name":"Standard_LRS","tier":1}}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestBuildJSONBodyStringValueWithEquals(t *testing.T) {
	got, err := buildJSONBody([]string{"filter=a=b"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"filter":"a=b"}`
	if string(got) != want {
		t.Fatalf("body = %q, want %q", string(got), want)
	}
}

func TestBuildJSONBodyInvalidRawJSON(t *testing.T) {
	if _, err := buildJSONBody(nil, []string{"count:=not-json"}); err == nil {
		t.Fatal("expected error for invalid raw JSON")
	}
}

func TestBuildJSONBodyInvalidFormats(t *testing.T) {
	cases := [][]string{
		{"missing-separator"},
		{"=value"},
	}
	for _, fields := range cases {
		if _, err := buildJSONBody(fields, nil); err == nil {
			t.Errorf("expected error for --json-field %v", fields)
		}
	}
	if _, err := buildJSONBody(nil, []string{"missing-separator"}); err == nil {
		t.Error("expected error for --json-field-raw without separator")
	}
}

func TestBuildJSONBodyNonObjectPathConflict(t *testing.T) {
	if _, err := buildJSONBody([]string{"sku=basic", "sku.name=x"}, nil); err == nil {
		t.Fatal("expected error when a scalar path is extended as an object")
	}
}

func TestBuildRequestOptions_JSONFieldsBody(t *testing.T) {
	svc := newTestService()
	cfg := config.Config{
		NoAuth:        true,
		JSONFields:    []string{"name=alpha"},
		JSONFieldsRaw: []string{"count:=2"},
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
	want := `{"count":2,"name":"alpha"}`
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
		{NoAuth: true, JSONFieldsRaw: []string{"a:=1"}, FormFields: []string{"b=2"}},
	}
	for _, cfg := range conflicts {
		if _, _, err := svc.BuildRequestOptions(cfg, "POST", "https://example.com/x"); err == nil {
			t.Errorf("expected conflict error for cfg %+v", cfg)
		}
	}
}

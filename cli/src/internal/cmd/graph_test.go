package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildGraphRequestBody(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		subscriptions    []string
		managementGroups []string
		top              int
		skip             int
		skipToken        string
		wantErr          bool
		check            func(t *testing.T, req graphRequest)
	}{
		{
			name:  "query only omits scope and options",
			query: "Resources | count",
			check: func(t *testing.T, req graphRequest) {
				if req.Query != "Resources | count" {
					t.Errorf("query = %q", req.Query)
				}
				if req.Subscriptions != nil {
					t.Errorf("subscriptions should be omitted, got %v", req.Subscriptions)
				}
				if req.ManagementGroups != nil {
					t.Errorf("managementGroups should be omitted, got %v", req.ManagementGroups)
				}
				if req.Options != nil {
					t.Errorf("options should be omitted, got %+v", req.Options)
				}
			},
		},
		{
			name:          "subscriptions are included",
			query:         "Resources | project name",
			subscriptions: []string{"sub-1", "sub-2"},
			check: func(t *testing.T, req graphRequest) {
				if len(req.Subscriptions) != 2 || req.Subscriptions[0] != "sub-1" {
					t.Errorf("subscriptions = %v", req.Subscriptions)
				}
			},
		},
		{
			name:             "management groups are included",
			query:            "Resources | project name",
			managementGroups: []string{"mg-1"},
			check: func(t *testing.T, req graphRequest) {
				if len(req.ManagementGroups) != 1 || req.ManagementGroups[0] != "mg-1" {
					t.Errorf("managementGroups = %v", req.ManagementGroups)
				}
			},
		},
		{
			name:      "paging fields populate options",
			query:     "Resources | project name",
			top:       5,
			skip:      10,
			skipToken: "abc",
			check: func(t *testing.T, req graphRequest) {
				if req.Options == nil {
					t.Fatalf("options should be present")
				}
				if req.Options.Top != 5 || req.Options.Skip != 10 || req.Options.SkipToken != "abc" {
					t.Errorf("options = %+v", req.Options)
				}
			},
		},
		{
			name:    "empty query is rejected",
			query:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := buildGraphRequestBody(tt.query, tt.subscriptions, tt.managementGroups, tt.top, tt.skip, tt.skipToken)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var req graphRequest
			if err := json.Unmarshal([]byte(body), &req); err != nil {
				t.Fatalf("body is not valid JSON: %v\nbody: %s", err, body)
			}
			if tt.check != nil {
				tt.check(t, req)
			}
		})
	}
}

func TestBuildGraphRequestBodyUsesDollarPrefixedOptionKeys(t *testing.T) {
	body, err := buildGraphRequestBody("Resources | count", nil, nil, 1, 2, "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	opts, ok := raw["options"]
	if !ok {
		t.Fatalf("options key missing in %s", body)
	}
	var optMap map[string]any
	if err := json.Unmarshal(opts, &optMap); err != nil {
		t.Fatalf("invalid options JSON: %v", err)
	}
	for _, key := range []string{"$top", "$skip", "$skipToken"} {
		if _, ok := optMap[key]; !ok {
			t.Errorf("expected option key %q in %s", key, string(opts))
		}
	}
}

func TestResolveGraphQueryFromArgument(t *testing.T) {
	got, err := resolveGraphQuery([]string{"Resources | count"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Resources | count" {
		t.Fatalf("query = %q", got)
	}
}

func TestResolveGraphQueryFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "query.kql")
	want := "Resources\n| project name, type\n"
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatalf("write query file: %v", err)
	}

	got, err := resolveGraphQuery(nil, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
}

func TestResolveGraphQueryRejectsArgumentAndFile(t *testing.T) {
	_, err := resolveGraphQuery([]string{"Resources | count"}, "query.kql")
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("expected combined input error, got %v", err)
	}
}

func TestResolveGraphQueryRejectsMissingQuery(t *testing.T) {
	_, err := resolveGraphQuery(nil, "")
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("expected missing query error, got %v", err)
	}
}

func TestResolveGraphQueryRejectsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.kql")
	if err := os.WriteFile(path, []byte(" \n\t"), 0o600); err != nil {
		t.Fatalf("write query file: %v", err)
	}

	_, err := resolveGraphQuery(nil, path)
	if err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("expected empty file error, got %v", err)
	}
}

func TestResolveGraphQueryReportsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.kql")
	_, err := resolveGraphQuery(nil, path)
	if err == nil || !strings.Contains(err.Error(), "failed to read --query-file") {
		t.Fatalf("expected missing file error, got %v", err)
	}
}

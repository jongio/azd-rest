package version

import "testing"

func TestInfo(t *testing.T) {
	if Info.Version == "" {
		t.Fatal("version should not be empty")
	}
	if Info.ExtensionID != "jongio.azd.rest" {
		t.Fatalf("expected extension ID jongio.azd.rest, got %s", Info.ExtensionID)
	}
}

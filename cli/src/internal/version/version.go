// Package version provides version information for the azd rest extension.
// Version information is set at build time via ldflags.
package version

// Version is the current version of the azd rest extension.
// It follows semantic versioning (e.g., "1.0.0").
// It is intended to be set at build time via ldflags:
//
//	go build -ldflags "-X github.com/jongio/azd-rest/cli/src/internal/version.Version=1.0.0"
var Version = "0.0.0-dev"

// BuildDate is the UTC timestamp of the build.
// It is intended to be set at build time via ldflags:
//
//	go build -ldflags "-X github.com/jongio/azd-rest/cli/src/internal/version.BuildDate=2025-01-09T12:00:00Z"
var BuildDate = "unknown"

// GitCommit is the git SHA used for the build.
// It is intended to be set at build time via ldflags:
//
//	go build -ldflags "-X github.com/jongio/azd-rest/cli/src/internal/version.GitCommit=abc123"
var GitCommit = "unknown"

// ExtensionID is the unique identifier for this extension.
// This ID is used by the azd extension registry and must match extension.yaml.
const ExtensionID = "jongio.azd.rest"

// Name is the human-readable name of the extension.
const Name = "azd rest"

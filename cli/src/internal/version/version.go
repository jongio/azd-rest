// Package version provides version information for the azd rest extension.
// Version information is set at build time via ldflags.
package version

import coreversion "github.com/jongio/azd-core/version"

// These variables are set at build time via ldflags:
//
//	go build -ldflags "-X github.com/jongio/azd-rest/src/internal/version.Version=1.0.0 -X github.com/jongio/azd-rest/src/internal/version.BuildDate=2025-01-09T12:00:00Z -X github.com/jongio/azd-rest/src/internal/version.GitCommit=abc123"
var Version = "0.0.0-dev"
var BuildDate = "unknown"
var GitCommit = "unknown"

// Info provides the shared version information for this extension.
var Info = coreversion.New("jongio.azd.rest", "azd rest")

func init() {
	Info.Version = Version
	Info.BuildDate = BuildDate
	Info.GitCommit = GitCommit
}

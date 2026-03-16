// Package version provides version information for the azd rest extension.
// Version information is set at build time via ldflags.
package version

import coreversion "github.com/jongio/azd-core/version"

// Version is the semantic version of the azd-rest extension, set at build time via ldflags.
var Version = "0.0.0-dev"

// BuildDate is the ISO 8601 build timestamp, set at build time via ldflags.
var BuildDate = "unknown"

// GitCommit is the short git commit SHA, set at build time via ldflags.
var GitCommit = "unknown"

// Info provides the shared version information for this extension.
var Info = coreversion.New("jongio.azd.rest", "azd rest")

func init() {
	Info.Version = Version
	Info.BuildDate = BuildDate
	Info.GitCommit = GitCommit
}

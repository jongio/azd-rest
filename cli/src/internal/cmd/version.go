package cmd

import (
	coreversion "github.com/jongio/azd-core/version"
	"github.com/jongio/azd-rest/src/internal/version"
	"github.com/spf13/cobra"
)

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	return coreversion.NewCommand(version.Info, &outputFormat)
}

package main

import (
	"fmt"
	"os"

	"github.com/jongio/azd-rest/cli/src/internal/cmd"
)

// version is set via ldflags during build
var version = "dev"

func main() {
	// Set version for command package
	cmd.SetVersion(version)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

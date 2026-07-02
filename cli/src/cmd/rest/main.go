// Package main provides the azd rest extension command-line interface.
// It enables execution of REST API calls with automatic Azure authentication.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jongio/azd-rest/src/internal/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		exitCode := 1
		var coder cmd.ExitCoder
		if errors.As(err, &coder) {
			exitCode = coder.ExitCode()
		}
		os.Exit(exitCode)
	}
}

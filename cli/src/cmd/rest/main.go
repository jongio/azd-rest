// Package main provides the azd rest extension command-line interface.
// It enables execution of REST API calls with automatic Azure authentication.
package main

import (
	"fmt"
	"os"

	"github.com/jongio/azd-rest/src/internal/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

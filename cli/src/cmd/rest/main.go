// Package main provides the azd rest extension command-line interface.
// It enables execution of REST API calls with automatic Azure authentication.
package main

import (
	"github.com/azure/azure-dev/cli/azd/pkg/azdext"

	"github.com/jongio/azd-rest/src/internal/cmd"
)

// main hands the whole extension lifecycle to the SDK. azdext.Run owns color
// setup, context creation, gRPC access-token injection, reserved-flag
// validation, structured error reporting back to the azd host, and the exit
// code. Anything this function does beyond building the root command would be
// duplication of framework behavior that upstream keeps evolving.
func main() {
	azdext.Run(cmd.NewRootCmd())
}

package cmd

import (
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/stretchr/testify/require"
)

// TestNoReservedFlagConflicts is a startup guard. azd reserves a set of global
// flags (things like --cwd, --debug, --no-prompt) for itself. If a subcommand
// here defines a flag with the same long name or shorthand, the extension still
// compiles and still passes every other test, but at runtime azd rejects the
// command tree and the extension is unusable.
//
// azdext.Run performs this same check before executing anything, so without
// this test the first person to find the problem is a user, not CI. Running it
// against the real root command means every subcommand added later is covered
// automatically, with no per-command test to remember to write.
func TestNoReservedFlagConflicts(t *testing.T) {
	require.NoError(t, azdext.ValidateNoReservedFlagConflicts(NewRootCmd()),
		"a command defines a flag that collides with an azd reserved global flag; "+
			"rename the extension flag, because the reserved names belong to the host")
}

// TestReservedFlagNamesIsNotEmpty guards against the check above silently
// becoming a no-op. If a future SDK release returned an empty reserved list,
// ValidateNoReservedFlagConflicts would pass unconditionally and the guard
// would look green while protecting nothing.
func TestReservedFlagNamesIsNotEmpty(t *testing.T) {
	require.NotEmpty(t, azdext.ReservedFlagNames(),
		"the SDK reported no reserved flags, which would make the conflict check vacuous")
}

package azdcore

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Verifies that azd-core is declared as a dependency in go.mod.
func TestAzdCoreDependencyPresent(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")

	dir := filepath.Dir(filename)
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(dir)))
	modPath := filepath.Join(moduleRoot, "go.mod")

	data, err := os.ReadFile(modPath)
	require.NoError(t, err, "go.mod should be readable")

	contents := string(data)
	require.True(t, strings.Contains(contents, "github.com/jongio/azd-core"), "go.mod must include azd-core as a dependency")
}

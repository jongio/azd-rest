package azdcore

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const requiredAzdCoreVersion = "github.com/jongio/azd-core v0.5.2-0.20260223042348-df3319c65059"

// Fails fast when go.mod drifts from the pinned azd-core version.
func TestAzdCoreVersionPinned(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")

	dir := filepath.Dir(filename)
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(dir)))
	modPath := filepath.Join(moduleRoot, "go.mod")

	data, err := os.ReadFile(modPath)
	require.NoError(t, err, "go.mod should be readable")

	contents := string(data)
	require.True(t, strings.Contains(contents, requiredAzdCoreVersion), "azd-core version must stay pinned to %s", requiredAzdCoreVersion)
}

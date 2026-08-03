package service

import (
	"io"
	"strings"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/require"
)

func withStdin(t *testing.T, body string) {
	t.Helper()
	previous := stdinReader
	stdinReader = strings.NewReader(body)
	t.Cleanup(func() { stdinReader = previous })
}

func TestBuildRequestOptions_DataFileStdin(t *testing.T) {
	withStdin(t, `{"name":"from-stdin"}`)

	opts, cleanup, err := newTestService().BuildRequestOptions(config.Config{
		NoAuth:   true,
		DataFile: "-",
	}, "POST", "https://example.com")
	require.NoError(t, err)
	defer cleanup()

	body, err := io.ReadAll(opts.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"from-stdin"}`, string(body))
}

func TestBuildRequestOptions_DataAtDashStdin(t *testing.T) {
	withStdin(t, `{"name":"from-data"}`)

	opts, cleanup, err := newTestService().BuildRequestOptions(config.Config{
		NoAuth: true,
		Data:   "@-",
	}, "POST", "https://example.com")
	require.NoError(t, err)
	defer cleanup()

	body, err := io.ReadAll(opts.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"from-data"}`, string(body))
}

func TestBuildRequestOptions_StdinKeepsBinaryBytes(t *testing.T) {
	raw := string([]byte{0x00, 0x01, 0x02, 0xff})
	withStdin(t, raw)

	opts, cleanup, err := newTestService().BuildRequestOptions(config.Config{
		NoAuth:   true,
		Binary:   true,
		DataFile: "-",
	}, "POST", "https://example.com")
	require.NoError(t, err)
	defer cleanup()

	body, err := io.ReadAll(opts.Body)
	require.NoError(t, err)
	require.Equal(t, []byte(raw), body)
}

func TestBuildRequestOptions_StdinAllowsEmptyBody(t *testing.T) {
	withStdin(t, "")

	opts, cleanup, err := newTestService().BuildRequestOptions(config.Config{
		NoAuth:   true,
		DataFile: "-",
	}, "POST", "https://example.com")
	require.NoError(t, err)
	defer cleanup()

	body, err := io.ReadAll(opts.Body)
	require.NoError(t, err)
	require.Empty(t, body)
}

func TestBuildRequestOptions_StdinConflictsWithData(t *testing.T) {
	withStdin(t, `{"name":"from-stdin"}`)

	_, _, err := newTestService().BuildRequestOptions(config.Config{
		NoAuth:   true,
		Data:     `{"name":"inline"}`,
		DataFile: "-",
	}, "POST", "https://example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be combined")
}

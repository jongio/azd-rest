package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-rest/src/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderXML_PrettyPrintsAndPreservesDeclaration(t *testing.T) {
	out, err := renderXML([]byte(`<?xml version="1.0" encoding="utf-8"?><EnumerationResults><Containers><Container><Name>logs</Name></Container></Containers></EnumerationResults>`))
	require.NoError(t, err)

	want := `<?xml version="1.0" encoding="utf-8"?>
<EnumerationResults>
  <Containers>
    <Container>
      <Name>logs</Name>
    </Container>
  </Containers>
</EnumerationResults>
`
	assert.Equal(t, want, out)
}

func TestRenderXML_InvalidXMLReturnsError(t *testing.T) {
	_, err := renderXML([]byte(`<root><broken></root>`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "xml format requires a valid XML response")
}

func TestExecute_XMLFormatWritesPrettyXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<root><item>one</item><item>two</item></root>`))
	}))
	defer server.Close()

	outputFile := filepath.Join(t.TempDir(), "out.xml")
	cfg := config.Defaults()
	cfg.NoAuth = true
	cfg.OutputFormat = "xml"
	cfg.OutputFile = outputFile

	err := newTestService().Execute(context.Background(), cfg, "GET", server.URL)
	require.NoError(t, err)

	out, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(out), "<root>\n  <item>one</item>\n  <item>two</item>\n</root>")
	assert.True(t, strings.HasSuffix(string(out), "\n"))
}

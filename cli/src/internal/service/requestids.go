package service

import (
	"fmt"
	"io"
	"net/http"

	"github.com/jongio/azd-rest/src/internal/client"
)

func writeRequestIDs(w io.Writer, headers http.Header) {
	for _, name := range []string{
		"x-ms-request-id",
		"x-ms-correlation-request-id",
		"x-ms-routing-request-id",
		"request-id",
		"client-request-id",
	} {
		value := headers.Get(name)
		if value == "" {
			continue
		}
		fmt.Fprintf(w, "%s: %s\n", name, client.RedactSensitiveHeader(name, value))
	}
}

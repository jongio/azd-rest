package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
)

type responseMetadata struct {
	Method       string              `json:"method"`
	URL          string              `json:"url"`
	Status       string              `json:"status"`
	StatusCode   int                 `json:"statusCode"`
	DurationMs   int64               `json:"durationMs"`
	SizeDownload int                 `json:"sizeDownload"`
	ContentType  string              `json:"contentType,omitempty"`
	Headers      map[string][]string `json:"headers"`
}

func newResponseMetadata(method, requestURL string, resp *client.Response) responseMetadata {
	return responseMetadata{
		Method:       method,
		URL:          requestURL,
		Status:       resp.Status,
		StatusCode:   resp.StatusCode,
		DurationMs:   durationMilliseconds(resp.Duration),
		SizeDownload: len(resp.Body),
		ContentType:  resp.Headers.Get("Content-Type"),
		Headers:      redactedMetadataHeaders(resp.Headers),
	}
}

func writeResponseMetadata(path, method, requestURL string, resp *client.Response) error {
	if path == "" {
		return nil
	}

	metadata := newResponseMetadata(method, requestURL, resp)
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode response metadata: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}
	return nil
}

func redactedMetadataHeaders(headers http.Header) map[string][]string {
	result := make(map[string][]string, len(headers))
	for name, values := range headers {
		redacted := make([]string, 0, len(values))
		for _, value := range values {
			redacted = append(redacted, client.RedactSensitiveHeader(name, value))
		}
		result[name] = redacted
	}
	return result
}

func durationMilliseconds(d time.Duration) int64 {
	return d.Milliseconds()
}

package service

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/config"
)

type dryRunOutput struct {
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Scope           string            `json:"scope,omitempty"`
	AuthMode        string            `json:"authMode"`
	Timeout         string            `json:"timeout"`
	MaxTime         string            `json:"maxTime,omitempty"`
	Retry           int               `json:"retry"`
	FollowRedirects bool              `json:"followRedirects"`
	MaxRedirects    int               `json:"maxRedirects"`
	MaxResponseSize int64             `json:"maxResponseSize"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            *dryRunBody       `json:"body,omitempty"`
}

type dryRunBody struct {
	Source string `json:"source"`
	Bytes  int64  `json:"bytes"`
}

func writeDryRun(w io.Writer, cfg config.Config, opts client.RequestOptions) error {
	body, err := dryRunBodyInfo(cfg, opts.Body)
	if err != nil {
		return err
	}

	output := dryRunOutput{
		Method:          opts.Method,
		URL:             opts.URL,
		Scope:           opts.Scope,
		AuthMode:        authMode(opts),
		Timeout:         cfg.Timeout.String(),
		Retry:           cfg.Retry,
		FollowRedirects: cfg.FollowRedirects,
		MaxRedirects:    cfg.MaxRedirects,
		MaxResponseSize: cfg.MaxResponseSize,
		Headers:         redactedHeaders(opts.Headers),
		Body:            body,
	}
	if cfg.MaxTime > 0 {
		output.MaxTime = cfg.MaxTime.String()
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func authMode(opts client.RequestOptions) string {
	if opts.SkipAuth {
		return "none"
	}
	return "bearer"
}

func redactedHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	redacted := make(map[string]string, len(headers))
	for _, key := range keys {
		redacted[key] = client.RedactSensitiveHeader(key, headers[key])
	}
	return redacted
}

func dryRunBodyInfo(cfg config.Config, body io.Reader) (*dryRunBody, error) {
	if body == nil {
		return nil, nil
	}

	size, err := bodySize(body)
	if err != nil {
		return nil, err
	}

	return &dryRunBody{
		Source: bodySource(cfg),
		Bytes:  size,
	}, nil
}

func bodySource(cfg config.Config) string {
	switch {
	case cfg.DataFile != "":
		return "data-file"
	case cfg.Data != "":
		return "data"
	case len(cfg.JSONFields) > 0 || len(cfg.JSONFieldsRaw) > 0:
		return "json-field"
	case len(cfg.FormFields) > 0:
		return "form-field"
	default:
		return "request-body"
	}
}

func bodySize(body io.Reader) (int64, error) {
	if seeker, ok := body.(io.Seeker); ok {
		current, err := seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, fmt.Errorf("failed to inspect request body: %w", err)
		}
		end, err := seeker.Seek(0, io.SeekEnd)
		if err != nil {
			return 0, fmt.Errorf("failed to inspect request body: %w", err)
		}
		if _, err := seeker.Seek(current, io.SeekStart); err != nil {
			return 0, fmt.Errorf("failed to restore request body: %w", err)
		}
		return end - current, nil
	}

	raw, err := io.ReadAll(body)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect request body: %w", err)
	}
	return int64(len(raw)), nil
}

var _ io.Reader = (*os.File)(nil)

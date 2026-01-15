package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jongio/azd-rest/src/internal/auth"
)

// RequestOptions contains options for HTTP requests
type RequestOptions struct {
	Method             string
	URL                string
	Body               io.Reader
	Headers            map[string]string
	Scope              string
	SkipAuth           bool
	Verbose            bool
	Timeout            time.Duration
	Insecure           bool
	FollowRedirects    bool
	MaxRedirects       int
	OutputFile         string
	Format             string
	TokenProvider      auth.TokenProvider
	Binary             bool
}

// Response contains HTTP response data
type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
}

// Client wraps HTTP client functionality
type Client struct {
	httpClient    *http.Client
	tokenProvider auth.TokenProvider
}

// NewClient creates a new HTTP client
func NewClient(tokenProvider auth.TokenProvider, insecure bool, timeout time.Duration) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		tokenProvider: tokenProvider,
	}
}

// Execute performs an HTTP request with the given options
func (c *Client) Execute(ctx context.Context, opts RequestOptions) (*Response, error) {
	startTime := time.Now()

	// Create request
	req, err := http.NewRequestWithContext(ctx, opts.Method, opts.URL, opts.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	// Add authentication if needed
	if !opts.SkipAuth && opts.Scope != "" && c.tokenProvider != nil {
		token, err := c.tokenProvider.GetToken(ctx, opts.Scope)
		if err != nil {
			return nil, fmt.Errorf("failed to get authentication token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Add default headers if not present
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "azd-rest/0.1.0 (azd extension)")
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	duration := time.Since(startTime)

	return &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       body,
		Duration:   duration,
	}, nil
}

// ShouldSkipAuth determines if authentication should be skipped
func ShouldSkipAuth(url string, headers map[string]string, skipAuth bool) bool {
	// Explicit skip flag
	if skipAuth {
		return true
	}

	// Check if Authorization header already present
	for key := range headers {
		if strings.EqualFold(key, "authorization") {
			return true
		}
	}

	// Check if URL is HTTP (not HTTPS)
	if strings.HasPrefix(strings.ToLower(url), "http://") {
		return true
	}

	return false
}

// DetectContentType attempts to determine if content is binary
func DetectContentType(body []byte, contentType string) bool {
	// If content-type header indicates binary
	binaryTypes := []string{
		"application/octet-stream",
		"application/pdf",
		"image/",
		"video/",
		"audio/",
	}

	for _, binType := range binaryTypes {
		if strings.Contains(strings.ToLower(contentType), binType) {
			return true
		}
	}

	// Check for binary content in body (simple heuristic)
	if len(body) > 0 {
		// If first 512 bytes contain null bytes, likely binary
		checkLen := 512
		if len(body) < checkLen {
			checkLen = len(body)
		}
		if bytes.ContainsRune(body[:checkLen], 0) {
			return true
		}
	}

	return false
}

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jongio/azd-rest/src/internal/auth"
	"github.com/jongio/azd-rest/src/internal/version"
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
	Retry              int
	MaxResponseSize    int64
	Paginate           bool
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
		// Use proxy from environment variables (HTTP_PROXY, HTTPS_PROXY, NO_PROXY)
		Proxy: http.ProxyFromEnvironment,
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

	// Configure redirect handling
	maxRedirects := opts.MaxRedirects
	if maxRedirects == 0 {
		maxRedirects = 10 // Default max redirects
	}

	originalClient := c.httpClient
	client := &http.Client{
		Transport: originalClient.Transport,
		Timeout:   originalClient.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !opts.FollowRedirects {
				return http.ErrUseLastResponse
			}
			// via contains all previous requests including the original, so len(via) is the redirect count
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		},
	}

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
		req.Header.Set("User-Agent", fmt.Sprintf("azd-rest/%s (azd extension)", version.Version))
	}

	// Log request details in verbose mode
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "> %s %s\n", opts.Method, opts.URL)
		for key, values := range req.Header {
			for _, value := range values {
				// Redact sensitive headers
				redactedValue := RedactSensitiveHeader(key, value)
				fmt.Fprintf(os.Stderr, "> %s: %s\n", key, redactedValue)
			}
		}
		fmt.Fprintf(os.Stderr, "> \n")
	}

	// Execute request with retry logic
	var resp *http.Response
	maxRetries := opts.Retry
	if maxRetries < 0 {
		maxRetries = 0
	}
	if maxRetries == 0 {
		maxRetries = 3 // Default retries
	}

	var lastErr error
	var bodyBytes []byte
	var bodyReader io.Reader = opts.Body

	// If body is provided and we might retry, read it into memory for retries
	// This allows us to retry with the same body content
	// Only do this if retries are actually enabled (not just default)
	if opts.Body != nil && maxRetries > 0 && opts.Retry > 0 {
		// Try to read body into memory (limit to 10MB for retry support)
		const maxBodySizeForRetry = 10 * 1024 * 1024
		limitedReader := io.LimitReader(opts.Body, maxBodySizeForRetry+1)
		var err error
		bodyBytes, err = io.ReadAll(limitedReader)
		if err == nil && int64(len(bodyBytes)) <= maxBodySizeForRetry {
			// Successfully read body into memory, can retry
			bodyReader = bytes.NewReader(bodyBytes)
		} else {
			// Body too large or read error - try seeker approach
			if seeker, ok := opts.Body.(io.Seeker); ok {
				if _, seekErr := seeker.Seek(0, io.SeekStart); seekErr == nil {
					bodyReader = opts.Body
				}
			}
			// If neither works, retries won't work properly (body consumed)
		}
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, etc.
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
			case <-time.After(backoff):
			}

			// Recreate request for retry
			if bodyReader != nil {
				// Reset body reader position if it's a bytes.Reader
				if br, ok := bodyReader.(*bytes.Reader); ok {
					br.Seek(0, io.SeekStart)
					req.Body = io.NopCloser(br)
					req.GetBody = func() (io.ReadCloser, error) {
						br.Seek(0, io.SeekStart)
						return io.NopCloser(br), nil
					}
				} else if seekerReader, ok := bodyReader.(interface {
					io.Reader
					io.Seeker
				}); ok {
					seekerReader.Seek(0, io.SeekStart)
					req.Body = io.NopCloser(seekerReader)
					req.GetBody = func() (io.ReadCloser, error) {
						seekerReader.Seek(0, io.SeekStart)
						return io.NopCloser(seekerReader), nil
					}
				} else {
					req.Body = io.NopCloser(bodyReader)
				}
			}
		}

		resp, lastErr = client.Do(req)
		if lastErr == nil {
			// Check if response status is retryable (5xx errors)
			if resp.StatusCode >= 500 && resp.StatusCode < 600 && attempt < maxRetries {
				resp.Body.Close()
				// Retry on server errors
				continue
			}
			break // Success or non-retryable error
		}

		// Check if error is retryable
		if !isRetryableError(lastErr) {
			return nil, fmt.Errorf("request failed: %w", lastErr)
		}

		// If this was the last attempt, return the error
		if attempt == maxRetries {
			return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
		}
	}

	if resp == nil {
		return nil, fmt.Errorf("request failed: %w", lastErr)
	}
	defer resp.Body.Close()

	// Read response body with size limit
	maxSize := opts.MaxResponseSize
	if maxSize <= 0 {
		maxSize = 100 * 1024 * 1024 // Default 100MB limit
	}

	limitedReader := io.LimitReader(resp.Body, maxSize)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if we hit the size limit
	if int64(len(responseBody)) >= maxSize {
		return nil, fmt.Errorf("response body exceeds maximum size of %d bytes", maxSize)
	}

	duration := time.Since(startTime)

	response := &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       responseBody,
		Duration:   duration,
	}

	// Handle pagination if enabled
	if opts.Paginate && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Use the same client configuration for pagination requests
		paginatedBody, err := handlePagination(ctx, client, opts, response)
		if err != nil {
			// If pagination fails, return original response
			// (pagination is best-effort, don't fail the whole request)
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Pagination failed: %v\n", err)
			}
			return response, nil
		}
		if paginatedBody != nil {
			response.Body = paginatedBody
		}
	}

	return response, nil
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

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Network errors are retryable
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"no such host",
		"network is unreachable",
		"temporary failure",
		"i/o timeout",
		"context deadline exceeded",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
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

// parseLinkHeader parses the Link header and extracts the next URL
func parseLinkHeader(linkHeader string) (string, bool) {
	if linkHeader == "" {
		return "", false
	}

	// Link header format: <url>; rel="next", <url>; rel="prev"
	// Regex to match <url> with rel="next"
	re := regexp.MustCompile(`<([^>]+)>;\s*rel=["']?next["']?`)
	matches := re.FindStringSubmatch(linkHeader)
	if len(matches) > 1 {
		return matches[1], true
	}

	// Also check for nextLink in Azure APIs (case-insensitive)
	reNext := regexp.MustCompile(`<([^>]+)>;\s*rel=["']?next["']?`)
	matches = reNext.FindStringSubmatch(strings.ToLower(linkHeader))
	if len(matches) > 1 {
		return matches[1], true
	}

	return "", false
}

// extractNextLinkFromBody extracts nextLink from JSON response body (Azure API format)
func extractNextLinkFromBody(body []byte) (string, bool) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", false
	}

	// Check for nextLink field (Azure Management API format)
	if nextLink, ok := data["nextLink"].(string); ok && nextLink != "" {
		return nextLink, true
	}

	// Check for @odata.nextLink (OData format)
	if nextLink, ok := data["@odata.nextLink"].(string); ok && nextLink != "" {
		return nextLink, true
	}

	// Check for next (Graph API format)
	if next, ok := data["@odata.next"].(string); ok && next != "" {
		return next, true
	}

	return "", false
}

// handlePagination handles pagination by following next links
func handlePagination(ctx context.Context, client *http.Client, opts RequestOptions, firstResponse *Response) ([]byte, error) {
	var allResults []interface{}
	var currentBody = firstResponse.Body
	var nextURL string
	var hasMore = true

	// Parse first response
	var firstData map[string]interface{}
	if err := json.Unmarshal(currentBody, &firstData); err != nil {
		// Not JSON, can't paginate
		return currentBody, nil
	}

	// Extract value array (Azure API format)
	if valueArray, ok := firstData["value"].([]interface{}); ok {
		allResults = append(allResults, valueArray...)
	} else {
		// If no value array, treat entire response as single item
		allResults = append(allResults, firstData)
	}

	// Check for next link in first response
	if next, ok := extractNextLinkFromBody(currentBody); ok {
		nextURL = next
		hasMore = true
	} else if linkHeader := firstResponse.Headers.Get("Link"); linkHeader != "" {
		if next, ok := parseLinkHeader(linkHeader); ok {
			nextURL = next
			hasMore = true
		}
	}

	// Follow pagination links (limit to prevent infinite loops)
	maxPages := 1000
	pageCount := 0

	// Only paginate if we found a next link
	for hasMore && nextURL != "" && pageCount < maxPages {
		pageCount++

		// Resolve relative URLs against the original request URL
		baseURL, err := url.Parse(opts.URL)
		if err != nil {
			break
		}
		nextURLParsed, err := url.Parse(nextURL)
		if err != nil {
			break
		}
		// Resolve relative URL
		resolvedURL := baseURL.ResolveReference(nextURLParsed).String()

		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "> Following pagination link: %s\n", resolvedURL)
		}

		// Create new request for next page
		req, err := http.NewRequestWithContext(ctx, opts.Method, resolvedURL, nil)
		if err != nil {
			break
		}

		// Copy headers from original request
		for key, value := range opts.Headers {
			req.Header.Set(key, value)
		}

		// Add authentication if needed
		if !opts.SkipAuth && opts.Scope != "" && opts.TokenProvider != nil {
			token, err := opts.TokenProvider.GetToken(ctx, opts.Scope)
			if err != nil {
				break
			}
			req.Header.Set("Authorization", "Bearer "+token)
		}

		// Add User-Agent
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", fmt.Sprintf("azd-rest/%s (azd extension)", version.Version))
		}

		// Execute request
		resp, err := client.Do(req)
		if err != nil {
			break
		}

		// Read response
		limitedReader := io.LimitReader(resp.Body, opts.MaxResponseSize)
		body, err := io.ReadAll(limitedReader)
		resp.Body.Close()

		if err != nil {
			break
		}

		// Parse response
		var pageData map[string]interface{}
		if err := json.Unmarshal(body, &pageData); err != nil {
			break
		}

		// Extract value array
		if valueArray, ok := pageData["value"].([]interface{}); ok {
			allResults = append(allResults, valueArray...)
		}

		// Check for next link
		nextURL = ""
		if next, ok := extractNextLinkFromBody(body); ok {
			nextURL = next
		} else if linkHeader := resp.Header.Get("Link"); linkHeader != "" {
			if next, ok := parseLinkHeader(linkHeader); ok {
				nextURL = next
			}
		}

		hasMore = (nextURL != "")
	}

	// Combine all results - always return combined if we have results
	// (even if only one page, we still want to remove nextLink)
	if len(allResults) > 0 {
		// Reconstruct response in Azure API format
		combined := map[string]interface{}{
			"value": allResults,
		}

		// Preserve other top-level fields from first response
		for key, value := range firstData {
			if key != "value" && key != "nextLink" && key != "@odata.nextLink" && key != "@odata.next" {
				combined[key] = value
			}
		}

		// Remove nextLink since we've paginated through everything
		delete(combined, "nextLink")
		delete(combined, "@odata.nextLink")
		delete(combined, "@odata.next")

		combinedJSON, err := json.Marshal(combined)
		if err != nil {
			return currentBody, err
		}

		return combinedJSON, nil
	}

	// If no results but we tried to paginate, return original
	return currentBody, nil
}

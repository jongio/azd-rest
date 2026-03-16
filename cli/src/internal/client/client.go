// Package client provides HTTP client functionality for the azd rest extension.
// It re-exports types and functions from github.com/jongio/azd-core/httpclient.
package client

import (
	"fmt"

	"github.com/jongio/azd-core/httpclient"
	"github.com/jongio/azd-rest/src/internal/version"
)

func init() {
	httpclient.UserAgent = fmt.Sprintf("azd-rest/%s (azd extension)", version.Version)
}

// Client is the HTTP client for making authenticated Azure REST API requests.
type Client = httpclient.Client

// RequestOptions configures individual HTTP request parameters.
type RequestOptions = httpclient.RequestOptions

// Response wraps an HTTP response with parsed body content.
type Response = httpclient.Response

// TokenProvider obtains OAuth tokens for authenticating HTTP requests.
type TokenProvider = httpclient.TokenProvider

// MockTokenProvider is a test double for TokenProvider.
type MockTokenProvider = httpclient.MockTokenProvider

// NewClient creates a new HTTP client configured for Azure REST API calls.
var NewClient = httpclient.NewClient

// ShouldSkipAuth determines whether authentication should be skipped for a given URL.
var ShouldSkipAuth = httpclient.ShouldSkipAuth

// DetectContentType infers the content type of the given request body.
var DetectContentType = httpclient.DetectContentType

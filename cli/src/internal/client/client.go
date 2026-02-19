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

// Re-export types from httpclient
type Client = httpclient.Client
type RequestOptions = httpclient.RequestOptions
type Response = httpclient.Response
type TokenProvider = httpclient.TokenProvider
type MockTokenProvider = httpclient.MockTokenProvider

// Re-export functions
var NewClient = httpclient.NewClient
var ShouldSkipAuth = httpclient.ShouldSkipAuth
var DetectContentType = httpclient.DetectContentType

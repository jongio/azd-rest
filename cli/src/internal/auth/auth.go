// Package auth re-exports authentication primitives from github.com/jongio/azd-core/auth.
package auth

import coreauth "github.com/jongio/azd-core/auth"

// Re-export types
type TokenProvider = coreauth.TokenProvider
type AzureTokenProvider = coreauth.AzureTokenProvider
type MockTokenProvider = coreauth.MockTokenProvider

// Re-export functions
var NewAzureTokenProvider = coreauth.NewAzureTokenProvider
var GetAzureToken = coreauth.GetAzureToken

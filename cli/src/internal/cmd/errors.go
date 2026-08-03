package cmd

// Error codes reported to the azd host through [azdext.LocalError.Code].
//
// The host renders these as the last segment of a telemetry name shaped
// `ext.<category>.<code>`, so they must be stable, lowercase, and snake_case.
// Treat them like a public API: renaming one breaks the continuity of any
// dashboard or alert built on it. Adding one is cheap; changing one is not.
//
// Keep the list alphabetical so additions do not collide in review.
const (
	// ErrCodeInvalidEnvDefault is reported when an AZD_REST_* environment
	// variable holds a value that the corresponding flag refuses to parse.
	ErrCodeInvalidEnvDefault = "invalid_env_default"
)

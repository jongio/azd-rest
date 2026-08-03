package service

import (
	"fmt"
	"net/url"
	"time"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

// hostFromURL extracts the host for telemetry. It returns "" for anything that
// does not parse, because a malformed host label is worse than no label: it
// would fragment the telemetry for a service under several spellings.
func hostFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Host
}

// Error codes reported to the azd host through the structured error types.
//
// The host renders a local error as `ext.<category>.<code>` in telemetry, so
// these strings are effectively a public contract. They are lowercase
// snake_case, stable, and alphabetical.
//
// These replace the old ExitCoder contract. azd does not propagate an
// extension's process exit code to the shell (only `azd exec` does that), so
// the exit codes those errors carried never reached a user. Structured errors
// do reach the user, both as a rendered suggestion and as classified telemetry.
const (
	// ErrCodeExpectUsage covers a malformed --expect argument, an invalid
	// JMESPath expression, or a response that is not JSON.
	ErrCodeExpectUsage = "invalid_expect_usage"

	// ErrCodeHTTPFail is reported when --fail is set and the response status is
	// 400 or higher.
	ErrCodeHTTPFail = "http_status_failure"

	// ErrCodeInvalidDataFormat covers an unknown --data-format value, a
	// conflict with the field-body flags, or a body that will not parse as YAML.
	ErrCodeInvalidDataFormat = "invalid_data_format"

	// ErrCodeInvalidMaxLatency is reported when --max-latency is not a positive
	// duration.
	ErrCodeInvalidMaxLatency = "invalid_max_latency"

	// ErrCodeMaxLatencyExceeded is reported when a response completes but takes
	// longer than the --max-latency budget.
	ErrCodeMaxLatencyExceeded = "max_latency_exceeded"

	// ErrCodeRawOutputUsage is reported when --raw-output is used without
	// --query.
	ErrCodeRawOutputUsage = "invalid_raw_output_usage"
)

// newUsageError builds the local error shape shared by every flag-usage
// failure in this package: the user passed something the command cannot act on,
// nothing was sent, and the fix is to change the invocation.
func newUsageError(code, message, suggestion string) error {
	return &azdext.LocalError{
		Message:    message,
		Code:       code,
		Category:   azdext.LocalErrorCategoryValidation,
		Suggestion: suggestion,
	}
}

// newDataFormatError reports invalid --data-format usage.
func newDataFormatError(err error) error {
	return newUsageError(
		ErrCodeInvalidDataFormat,
		err.Error(),
		fmt.Sprintf("--data-format accepts %q or %q. YAML bodies cannot be combined with --form-field, --json-field, or --json-field-raw.",
			dataFormatJSON, dataFormatYAML),
	)
}

// newExpectUsageError reports invalid --expect usage. This is deliberately
// distinct from an assertion that simply did not hold: a failed assertion means
// the command worked and the response was wrong, which is not a usage error.
func newExpectUsageError(message string) error {
	return newUsageError(
		ErrCodeExpectUsage,
		message,
		"--expect takes a JMESPath expression, optionally followed by =value, and requires a JSON response.",
	)
}

// newRawOutputUsageError reports --raw-output used without --query.
func newRawOutputUsageError(message string) error {
	return newUsageError(
		ErrCodeRawOutputUsage,
		message,
		"Add a --query expression, or drop --raw-output to print the whole JSON response.",
	)
}

// newMaxLatencyConfigError reports an unparseable --max-latency value.
func newMaxLatencyConfigError(value string) error {
	return newUsageError(
		ErrCodeInvalidMaxLatency,
		fmt.Sprintf("invalid --max-latency %q: use a positive duration such as 500ms or 2s", value),
		"Pass a positive Go duration, for example 500ms, 2s, or 1m.",
	)
}

// newHTTPFailError reports an error status under --fail. The response body has
// already been written by the time this is returned, so the details the user
// needs are on screen; this error only classifies the outcome.
func newHTTPFailError(status int, serviceName string) error {
	return &azdext.ServiceError{
		Message:     fmt.Sprintf("request failed with HTTP %d (--fail)", status),
		ErrorCode:   ErrCodeHTTPFail,
		StatusCode:  status,
		ServiceName: serviceName,
		Suggestion:  "The response body above carries the service's own error detail. Drop --fail if a non-2xx status is an expected outcome for this call.",
	}
}

// newMaxLatencyExceededError reports a response that completed but overran the
// --max-latency budget. It is a service error rather than a usage error because
// the request was well formed and the remote service was simply slow.
func newMaxLatencyExceededError(budget, actual time.Duration, status int, serviceName string) error {
	return &azdext.ServiceError{
		Message:     fmt.Sprintf("response took %s, over the --max-latency budget of %s", actual, budget),
		ErrorCode:   ErrCodeMaxLatencyExceeded,
		StatusCode:  status,
		ServiceName: serviceName,
		Suggestion:  "Raise --max-latency if the budget is too tight, or investigate why the service is slower than expected.",
	}
}

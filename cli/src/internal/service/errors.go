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
	// ErrCodeAllowStatus is reported when --allow-status is not a status code
	// or range inside 100-599.
	ErrCodeAllowStatus = "invalid_allow_status"

	// ErrCodeCacheConfig is reported when --cache-ttl is not a valid,
	// non-negative duration.
	ErrCodeCacheConfig = "invalid_cache_config"

	// ErrCodeCountUsage covers --count combined with a flag that suppresses the
	// body it needs to count.
	ErrCodeCountUsage = "invalid_count_usage"

	// ErrCodeDiffUsage covers a missing or unreadable --diff baseline, a
	// baseline that is not valid JSON, or a non-JSON response.
	ErrCodeDiffUsage = "invalid_diff_usage"

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

	// ErrCodeTemplateConfig covers a missing or unreadable --template file and
	// a template that will not parse.
	ErrCodeTemplateConfig = "invalid_template_config"

	// ErrCodeValidateSchemaUsage covers a missing or unreadable
	// --validate-schema file, a file that is not a valid JSON Schema, or a
	// response that is not JSON.
	ErrCodeValidateSchemaUsage = "invalid_validate_schema_usage"
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

// newAllowStatusError reports an unusable --allow-status value.
func newAllowStatusError(message string) error {
	return newUsageError(
		ErrCodeAllowStatus,
		message,
		"--allow-status takes a comma-separated list of status codes or ranges inside 100-599, for example \"404\" or \"200-204, 404\".",
	)
}

// newCountUsageError reports --count combined with a flag that suppresses the
// response body it needs.
func newCountUsageError(message string) error {
	return newUsageError(
		ErrCodeCountUsage,
		message,
		"--count needs the response body, so it cannot be combined with --no-body or --template.",
	)
}

// newDiffUsageError reports invalid --diff usage: a baseline that cannot be
// read or parsed, or a response that is not JSON. A baseline that reads fine
// but does not match is a drift result, not a usage error, so it is reported
// separately.
func newDiffUsageError(message string) error {
	return newUsageError(
		ErrCodeDiffUsage,
		message,
		"--diff takes a path to a JSON baseline file and requires a JSON response.",
	)
}

// newTemplateConfigError reports a --template file that cannot be read or
// parsed.
func newTemplateConfigError(message string) error {
	return newUsageError(
		ErrCodeTemplateConfig,
		message,
		"--template takes a path to a Go text/template file. Check the path and the template syntax.",
	)
}

// newCacheConfigError reports an unusable --cache-ttl value.
func newCacheConfigError(message string) error {
	return newUsageError(
		ErrCodeCacheConfig,
		message,
		"--cache-ttl takes a non-negative Go duration, for example 30s, 5m, or 1h.",
	)
}

// newValidateSchemaUsageError reports invalid --validate-schema usage. A
// response that reads fine but does not conform to the schema is a conformance
// failure, not a usage error, so it is reported separately.
func newValidateSchemaUsageError(message string) error {
	return newUsageError(
		ErrCodeValidateSchemaUsage,
		message,
		"--validate-schema takes a path to a JSON Schema file and requires a JSON response.",
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

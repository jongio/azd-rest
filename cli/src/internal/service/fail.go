package service

import "fmt"

// httpFailExitCode is the process exit code returned when --fail is set and the
// response status is 400 or higher. It matches curl's --fail exit code (22).
const httpFailExitCode = 22

// httpFailError signals that --fail was set and the response returned an error
// status. It implements the ExitCoder contract (Error and ExitCode) so main can
// translate it into exit code 22 instead of the generic exit code 1. The
// response body is written before this error is returned, so error details
// remain visible.
type httpFailError struct {
	status int
}

func (e *httpFailError) Error() string {
	return fmt.Sprintf("request failed with HTTP %d (--fail)", e.status)
}

// ExitCode returns 22 for a failed request under --fail.
func (e *httpFailError) ExitCode() int { return httpFailExitCode }

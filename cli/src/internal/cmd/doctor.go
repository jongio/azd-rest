// Package cmd provides CLI commands for the azd rest extension.
package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jongio/azd-core/auth"
	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/service"
	"github.com/spf13/cobra"
)

// doctorManagementScope is the scope used to verify that authentication works.
const doctorManagementScope = "https://management.azure.com/.default"

// doctorTokenProviderFactory creates the token provider used by doctor. It is a
// variable so tests can inject a mock provider.
var doctorTokenProviderFactory = service.DefaultTokenProviderFactory

// doctor check names and the JSON output format value.
const (
	formatJSON      = "json"
	checkNameScope  = "Scope detection"
	checkNameAuth   = "Azure authentication"
	checkNameClaims = "Token claims"
)

// checkStatus is the outcome of a single diagnostic check.
type checkStatus string

const (
	statusOK   checkStatus = "ok"
	statusWarn checkStatus = "warn"
	statusFail checkStatus = "fail"
)

// doctorCheck is the result of one diagnostic check.
type doctorCheck struct {
	Name        string      `json:"name"`
	Status      checkStatus `json:"status"`
	Detail      string      `json:"detail,omitempty"`
	Remediation string      `json:"remediation,omitempty"`
}

// NewDoctorCommand returns the doctor subcommand, which runs a set of checks to
// confirm azd rest can authenticate to Azure before you make real calls.
func NewDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose authentication and configuration issues",
		Long: `Run a set of checks to confirm azd rest can authenticate to Azure and
detect scopes. Use it when a request fails with an auth error and you want to
know whether the problem is your credentials, your scope, or something else.

Exits non-zero if any check fails.`,
		Example: `  # Run all checks
  azd rest doctor

  # Machine-readable output
  azd rest doctor --format json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			tp, tpErr := doctorTokenProviderFactory()
			return runDoctor(ctx, tp, tpErr, outputFormat, cmd.OutOrStdout())
		},
	}
}

// runDoctor executes the diagnostic checks, writes the report, and returns an
// error when any check fails so scripts get a non-zero exit code.
func runDoctor(ctx context.Context, tp client.TokenProvider, tpErr error, format string, out io.Writer) error {
	checks := []doctorCheck{checkScopeDetection()}

	authCheck, token := checkAuthentication(ctx, tp, tpErr)
	checks = append(checks, authCheck)
	if token != "" {
		checks = append(checks, checkTokenClaims(token))
	}

	if format == formatJSON {
		if err := writeDoctorJSON(out, checks); err != nil {
			return err
		}
	} else {
		writeDoctorText(out, checks)
	}

	failures := 0
	for _, c := range checks {
		if c.Status == statusFail {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("doctor found %d issue(s); see output above", failures)
	}
	return nil
}

// checkScopeDetection verifies that scope detection resolves a management URL.
// It makes no network call.
func checkScopeDetection() doctorCheck {
	const sampleURL = "https://management.azure.com/subscriptions?api-version=2020-01-01"
	scope, err := auth.DetectScope(sampleURL)
	if err != nil || scope == "" {
		return doctorCheck{
			Name:        checkNameScope,
			Status:      statusFail,
			Detail:      "could not detect a scope for management.azure.com",
			Remediation: "Pass --scope explicitly when calling a service.",
		}
	}
	return doctorCheck{
		Name:   checkNameScope,
		Status: statusOK,
		Detail: fmt.Sprintf("management.azure.com resolves to %s", scope),
	}
}

// checkAuthentication verifies a token can be acquired. It returns the token so
// the caller can inspect its claims.
func checkAuthentication(ctx context.Context, tp client.TokenProvider, tpErr error) (doctorCheck, string) {
	const remediation = "Run 'az login', or set AZURE_CLIENT_ID, AZURE_TENANT_ID, and AZURE_CLIENT_SECRET."

	if tpErr != nil || tp == nil {
		detail := "could not create an Azure credential"
		if tpErr != nil {
			detail = tpErr.Error()
		}
		return doctorCheck{Name: checkNameAuth, Status: statusFail, Detail: detail, Remediation: remediation}, ""
	}

	token, err := tp.GetToken(ctx, doctorManagementScope)
	if err != nil {
		return doctorCheck{Name: checkNameAuth, Status: statusFail, Detail: err.Error(), Remediation: remediation}, ""
	}
	return doctorCheck{
		Name:   checkNameAuth,
		Status: statusOK,
		Detail: "acquired a token for https://management.azure.com",
	}, token
}

// checkTokenClaims decodes the acquired token and reports the tenant and
// expiry. A token that cannot be decoded, or that has expired, is a warning
// rather than a failure since the acquisition itself succeeded.
func checkTokenClaims(token string) doctorCheck {
	claims, ok := decodeTokenClaims(token)
	if !ok {
		return doctorCheck{Name: checkNameClaims, Status: statusWarn, Detail: "token acquired but its claims could not be decoded"}
	}

	if !claims.Expiry.IsZero() && time.Now().After(claims.Expiry) {
		return doctorCheck{
			Name:        checkNameClaims,
			Status:      statusWarn,
			Detail:      fmt.Sprintf("token expired at %s", claims.Expiry.UTC().Format(time.RFC3339)),
			Remediation: "Run 'az login' to refresh your credentials.",
		}
	}

	var parts []string
	if claims.TenantID != "" {
		parts = append(parts, "tenant "+claims.TenantID)
	}
	if !claims.Expiry.IsZero() {
		parts = append(parts, "expires "+claims.Expiry.UTC().Format(time.RFC3339))
	}
	return doctorCheck{Name: checkNameClaims, Status: statusOK, Detail: strings.Join(parts, ", ")}
}

// tokenClaims holds the subset of JWT claims doctor reports on.
type tokenClaims struct {
	TenantID string
	Expiry   time.Time
}

// decodeTokenClaims does a best-effort local decode of a JWT payload. It never
// returns the raw token and does not verify the signature.
func decodeTokenClaims(token string) (tokenClaims, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return tokenClaims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return tokenClaims{}, false
	}
	var raw struct {
		TID string `json:"tid"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return tokenClaims{}, false
	}
	claims := tokenClaims{TenantID: raw.TID}
	if raw.Exp > 0 {
		claims.Expiry = time.Unix(raw.Exp, 0)
	}
	return claims, true
}

// writeDoctorText writes the checks as a human-readable report.
func writeDoctorText(out io.Writer, checks []doctorCheck) {
	symbols := map[checkStatus]string{
		statusOK:   "[ ok ]",
		statusWarn: "[warn]",
		statusFail: "[fail]",
	}
	for _, c := range checks {
		fmt.Fprintf(out, "%s %s\n", symbols[c.Status], c.Name)
		if c.Detail != "" {
			fmt.Fprintf(out, "       %s\n", c.Detail)
		}
		if c.Remediation != "" {
			fmt.Fprintf(out, "       fix: %s\n", c.Remediation)
		}
	}
}

// writeDoctorJSON writes the checks as indented JSON.
func writeDoctorJSON(out io.Writer, checks []doctorCheck) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(checks)
}

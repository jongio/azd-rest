package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// NewJWTCommand returns the jwt subcommand, which decodes an access token and
// prints its claims locally without verifying the signature or making any
// network call.
func NewJWTCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "jwt <token>",
		Short: "Decode a JWT access token and show its claims",
		Long: `Decode a JWT access token locally and print its claims.

jwt reads the token from its argument, base64-decodes the claims segment, and
prints the tenant, object ID, app ID, audience, username, granted scopes, roles,
and expiry. With --format json it prints the full set of decoded claims.

The signature is not verified and no network request is made. The raw token is
never written to output. This is useful for inspecting a token from
'az account get-access-token', an Authorization header, or a teammate.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJWT(args[0], outputFormat, cmd.OutOrStdout())
		},
	}
}

// runJWT decodes the token's claims and writes them to out. It is separated
// from the cobra command so tests can exercise it directly.
func runJWT(token, format string, out io.Writer) error {
	claims, err := decodeJWTClaims(token)
	if err != nil {
		return fmt.Errorf("failed to decode token: %w", err)
	}

	if strings.EqualFold(format, "json") {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(claims)
	}

	writeIdentityText(out, claimsToIdentity(claims))
	return nil
}

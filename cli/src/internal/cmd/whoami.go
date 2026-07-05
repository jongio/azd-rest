package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jongio/azd-rest/src/internal/client"
	"github.com/jongio/azd-rest/src/internal/service"
	"github.com/spf13/cobra"
)

// managementScope is the default OAuth scope used to acquire a token for
// identity inspection when no --scope override is provided.
const managementScope = "https://management.azure.com/.default"

// whoamiTokenProviderFactory builds the token provider used by the whoami
// command. It is a package variable so tests can inject a stub provider.
var whoamiTokenProviderFactory = service.DefaultTokenProviderFactory

// identity holds the subset of token claims that describe the caller.
type identity struct {
	Tenant    string    `json:"tenant,omitempty"`
	ObjectID  string    `json:"objectId,omitempty"`
	AppID     string    `json:"appId,omitempty"`
	Audience  string    `json:"audience,omitempty"`
	Username  string    `json:"username,omitempty"`
	Scopes    []string  `json:"scopes,omitempty"`
	Roles     []string  `json:"roles,omitempty"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

// NewWhoamiCommand returns the whoami subcommand.
func NewWhoamiCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the authenticated Azure identity",
		Long: `Show the Azure identity behind the credentials azd rest would use.

whoami acquires a token for the Azure Resource Manager scope (or the scope
given with --scope), decodes the token locally, and prints the tenant, object
ID, app ID, audience, granted scopes, and expiry. The raw token is never
printed and no request other than token acquisition is made.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			tp, err := whoamiTokenProviderFactory()
			if err != nil {
				return fmt.Errorf("failed to create token provider: %w", err)
			}
			tokenScope := scope
			if tokenScope == "" {
				tokenScope = managementScope
			}
			return runWhoami(ctx, tp, tokenScope, outputFormat, cmd.OutOrStdout())
		},
	}
}

// runWhoami acquires a token for the given scope, decodes its claims, and
// writes the identity to out. It is separated from the cobra command so tests
// can inject a token provider.
func runWhoami(ctx context.Context, tp client.TokenProvider, tokenScope, format string, out io.Writer) error {
	token, err := tp.GetToken(ctx, tokenScope)
	if err != nil {
		return err
	}
	claims, err := decodeJWTClaims(token)
	if err != nil {
		return fmt.Errorf("failed to decode token: %w", err)
	}
	id := claimsToIdentity(claims)

	if strings.EqualFold(format, "json") {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(id)
	}
	writeIdentityText(out, id)
	return nil
}

// decodeJWTClaims decodes the claims (payload) segment of a JWT into a map.
// It does not verify the signature; it only reads the caller-visible claims.
func decodeJWTClaims(token string) (map[string]any, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("not a JWT: expected 3 dot-separated segments, got %d", len(parts))
	}
	payload, err := decodeJWTSegment(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to base64-decode claims: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims JSON: %w", err)
	}
	return claims, nil
}

// decodeJWTSegment decodes a JWT segment. Per RFC 7519 segments are base64url
// without padding, but padded input is tolerated as well.
func decodeJWTSegment(seg string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(seg); err == nil {
		return b, nil
	}
	if m := len(seg) % 4; m != 0 {
		seg += strings.Repeat("=", 4-m)
	}
	return base64.URLEncoding.DecodeString(seg)
}

// claimsToIdentity maps raw JWT claims onto the identity fields azd rest shows.
func claimsToIdentity(claims map[string]any) identity {
	id := identity{
		Tenant:   claimString(claims, "tid"),
		ObjectID: claimString(claims, "oid"),
		Audience: audienceString(claims["aud"]),
		AppID:    firstNonEmpty(claimString(claims, "appid"), claimString(claims, "azp")),
		Username: firstNonEmpty(
			claimString(claims, "upn"),
			claimString(claims, "preferred_username"),
			claimString(claims, "unique_name"),
			claimString(claims, "name"),
		),
	}
	if scp := claimString(claims, "scp"); scp != "" {
		id.Scopes = strings.Fields(scp)
	}
	id.Roles = stringSlice(claims["roles"])
	if exp, ok := toFloat(claims["exp"]); ok {
		id.ExpiresAt = time.Unix(int64(exp), 0)
	}
	return id
}

func writeIdentityText(out io.Writer, id identity) {
	rows := [][2]string{
		{"Tenant", id.Tenant},
		{"Object ID", id.ObjectID},
		{"App ID", id.AppID},
		{"Username", id.Username},
		{"Audience", id.Audience},
	}
	for _, r := range rows {
		if r[1] != "" {
			fmt.Fprintf(out, "%-11s %s\n", r[0]+":", r[1])
		}
	}
	if len(id.Scopes) > 0 {
		fmt.Fprintf(out, "%-11s %s\n", "Scopes:", strings.Join(id.Scopes, " "))
	}
	if len(id.Roles) > 0 {
		fmt.Fprintf(out, "%-11s %s\n", "Roles:", strings.Join(id.Roles, " "))
	}
	if !id.ExpiresAt.IsZero() {
		fmt.Fprintf(out, "%-11s %s\n", "Expires:", id.ExpiresAt.Local().Format(time.RFC3339))
	}
}

func claimString(claims map[string]any, key string) string {
	if v, ok := claims[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// audienceString returns the audience claim as a string. The aud claim may be
// a single string or an array of strings; the first value is used for arrays.
func audienceString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []any:
		if len(val) > 0 {
			if s, ok := val[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

func stringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func toFloat(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

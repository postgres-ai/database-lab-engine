/*
2020 © Postgres.ai
*/

// Package commands provides general resources for CLI.
package commands

import (
	"net/url"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi"
)

// CLI configuration keys.
const (
	EnvironmentIDKey  = "environment-id"
	URLKey            = "url"
	TokenKey          = "token"
	InsecureKey       = "insecure"
	RequestTimeoutKey = "request-timeout"
	FwServerURLKey    = "forwarding-server-url"
	FwLocalPortKey    = "forwarding-local-port"
	IdentityFileKey   = "identity-file"
)

// ClientByCLIContext creates a new Database Lab API client.
func ClientByCLIContext(cliCtx *cli.Context) (*dblabapi.Client, error) {
	remoteURL, err := url.Parse(cliCtx.String(URLKey))
	if err != nil {
		return nil, err
	}

	if cliCtx.String(FwServerURLKey) != "" && cliCtx.String(FwLocalPortKey) != "" {
		remoteURL.Host = BuildHostname(remoteURL.Hostname(), cliCtx.String(FwLocalPortKey))
	}

	options := dblabapi.Options{
		Host:              remoteURL.String(),
		VerificationToken: cliCtx.String(TokenKey),
		Insecure:          cliCtx.Bool(InsecureKey),
		RequestTimeout:    cliCtx.Duration(RequestTimeoutKey),
	}

	// TODO(akartasov): Init and use logger.
	return dblabapi.NewClient(options)
}

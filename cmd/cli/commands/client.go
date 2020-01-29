/*
2020 © Postgres.ai
*/

// Package commands provides general resources for CLI.
package commands

import (
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/pkg/client/dblabapi"
)

// CLI configuration keys.
const (
	EnvironmentIDKey = "environment_id"
	URLKey           = "url"
	TokenKey         = "token"
	InsecureKey      = "insecure"
)

// ClientByCLIContext creates a new Database Lab API client.
func ClientByCLIContext(cliCtx *cli.Context) (*dblabapi.Client, error) {
	options := dblabapi.Options{
		Host:              cliCtx.String(URLKey),
		VerificationToken: cliCtx.String(TokenKey),
		Insecure:          cliCtx.Bool(InsecureKey),
	}

	// TODO(akartasov): Init and use logger.
	return dblabapi.NewClient(options, nil)
}

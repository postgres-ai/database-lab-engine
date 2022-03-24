/*
2020 © Postgres.ai
*/

package physical

import (
	"fmt"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/defaults"
)

const (
	pgbackrestTool = "pgbackrest"
)

// pgbackrest defines a pgBackRest as an archival restoration tool.
type pgbackrest struct {
	options pgbackrestOptions
}

type pgbackrestOptions struct {
	Stanza    string `yaml:"stanza"`
	ForceInit bool   `yaml:"forceInit"`
}

func newPgBackRest(options pgbackrestOptions) *pgbackrest {
	return &pgbackrest{
		options: options,
	}
}

// GetRestoreCommand returns a command to restore data.
func (p *pgbackrest) GetRestoreCommand() string {
	restoreCmd := fmt.Sprintf("sudo -Eu postgres pgbackrest --type=standby --pg1-path=${PGDATA} --stanza=%[1]s restore "+
		"--recovery-option=restore_command='pgbackrest --pg1-path=${PGDATA} --stanza=%[1]s archive-get %%f %%p'", p.options.Stanza)

	if p.options.ForceInit {
		restoreCmd += " --delta"
	}

	return restoreCmd
}

// GetRecoveryConfig returns a recovery config to restore data.
func (p *pgbackrest) GetRecoveryConfig(pgVersion float64) map[string]string {
	recoveryCfg := map[string]string{
		"restore_command": fmt.Sprintf("pgbackrest --pg1-path=${PGDATA} --stanza=%s archive-get %%f %%p", p.options.Stanza),
	}

	if pgVersion < defaults.PGVersion12 {
		recoveryCfg["recovery_target_timeline"] = "latest"
	}

	return recoveryCfg
}

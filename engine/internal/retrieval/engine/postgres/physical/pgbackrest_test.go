package physical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPgBackRestRecoveryConfig(t *testing.T) {
	pgbackrest := newPgBackRest(pgbackrestOptions{Stanza: "stanzaName"})

	recoveryConfig := pgbackrest.GetRecoveryConfig(11.7)
	expectedResponse11 := map[string]string{
		"restore_command":          "pgbackrest --pg1-path=${PGDATA} --stanza=stanzaName archive-get %f %p",
		"recovery_target_timeline": "latest",
	}
	assert.Equal(t, expectedResponse11, recoveryConfig)

	recoveryConfig = pgbackrest.GetRecoveryConfig(12.3)
	expectedResponse12 := map[string]string{
		"restore_command": "pgbackrest --pg1-path=${PGDATA} --stanza=stanzaName archive-get %f %p",
	}
	assert.Equal(t, expectedResponse12, recoveryConfig)
}

func TestPgBackRestRestoreCommand(t *testing.T) {
	pgbackrest := newPgBackRest(pgbackrestOptions{Stanza: "stanzaName"})

	restoreCmd := pgbackrest.GetRestoreCommand()
	expectedResponse := "sudo -Eu postgres pgbackrest --type=standby --pg1-path=${PGDATA} --stanza=stanzaName restore " +
		"--recovery-option=restore_command='pgbackrest --pg1-path=${PGDATA} --stanza=stanzaName archive-get %f %p'"
	assert.Equal(t, expectedResponse, restoreCmd)

	pgbackrest.options.Delta = true
	restoreCmd = pgbackrest.GetRestoreCommand()
	expectedResponse = "sudo -Eu postgres pgbackrest --type=standby --pg1-path=${PGDATA} --stanza=stanzaName restore " +
		"--recovery-option=restore_command='pgbackrest --pg1-path=${PGDATA} --stanza=stanzaName archive-get %f %p' --delta"
	assert.Equal(t, expectedResponse, restoreCmd)
}

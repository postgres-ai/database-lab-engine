package physical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPgBackRestRecoveryConfig(t *testing.T) {
	pgbackrest := newPgBackRest("dataDir", pgbackrestOptions{Stanza: "stanzaName"})

	recoveryConfig := pgbackrest.GetRecoveryConfig(11.7)
	expectedResponse11 := map[string]string{
		"restore_command":          "pgbackrest --pg1-path=dataDir --stanza=stanzaName archive-get %f %p",
		"recovery_target_timeline": "latest",
	}
	assert.Equal(t, expectedResponse11, recoveryConfig)

	recoveryConfig = pgbackrest.GetRecoveryConfig(12.3)
	expectedResponse12 := map[string]string{
		"restore_command": "pgbackrest --pg1-path=dataDir --stanza=stanzaName archive-get %f %p",
	}
	assert.Equal(t, expectedResponse12, recoveryConfig)
}

func TestPgBackRestRestoreCommand(t *testing.T) {
	pgbackrest := newPgBackRest("dataDir", pgbackrestOptions{Stanza: "stanzaName"})

	restoreCmd := pgbackrest.GetRestoreCommand()
	expectedResponse := "sudo -Eu postgres pgbackrest --type=standby --pg1-path=dataDir --stanza=stanzaName restore"
	assert.Equal(t, expectedResponse, restoreCmd)

	pgbackrest.options.ForceInit = true
	restoreCmd = pgbackrest.GetRestoreCommand()
	expectedResponse = "sudo -Eu postgres pgbackrest --delta --type=standby --pg1-path=dataDir --stanza=stanzaName restore"
	assert.Equal(t, expectedResponse, restoreCmd)
}

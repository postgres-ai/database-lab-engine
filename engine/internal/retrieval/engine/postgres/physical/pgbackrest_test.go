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

func TestPgBackRestRestoreCommand_DifferentStanzas(t *testing.T) {
	testCases := []struct {
		name     string
		stanza   string
		delta    bool
		hasDelta bool
	}{
		{name: "production stanza without delta", stanza: "prod-db", delta: false, hasDelta: false},
		{name: "production stanza with delta", stanza: "prod-db", delta: true, hasDelta: true},
		{name: "staging stanza", stanza: "staging", delta: false, hasDelta: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := newPgBackRest(pgbackrestOptions{Stanza: tc.stanza, Delta: tc.delta})
			cmd := p.GetRestoreCommand()
			assert.Contains(t, cmd, "--stanza="+tc.stanza)
			assert.Contains(t, cmd, "--type=standby")
			assert.Contains(t, cmd, "--pg1-path=${PGDATA}")
			if tc.hasDelta {
				assert.Contains(t, cmd, "--delta")
			} else {
				assert.NotContains(t, cmd, "--delta")
			}
		})
	}
}

func TestPgBackRestRestoreCommand_CustomOptions(t *testing.T) {
	baseCmd := "sudo -Eu postgres pgbackrest --type=standby --pg1-path=${PGDATA} --stanza=stanzaName restore " +
		"--recovery-option=restore_command='pgbackrest --pg1-path=${PGDATA} --stanza=stanzaName archive-get %f %p'"

	testCases := []struct {
		name          string
		delta         bool
		customOptions []string
		expected      string
	}{
		{name: "no custom options", customOptions: nil, expected: baseCmd},
		{name: "empty custom options", customOptions: []string{}, expected: baseCmd},
		{name: "single option", customOptions: []string{"--db-include=mydb"}, expected: baseCmd + " --db-include=mydb"},
		{name: "multiple options", customOptions: []string{"--db-include=mydb", "--process-max=4"}, expected: baseCmd + " --db-include=mydb --process-max=4"},
		{name: "appended after delta", delta: true, customOptions: []string{"--db-include=mydb"}, expected: baseCmd + " --delta --db-include=mydb"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := newPgBackRest(pgbackrestOptions{Stanza: "stanzaName", Delta: tc.delta, CustomOptions: tc.customOptions})
			assert.Equal(t, tc.expected, p.GetRestoreCommand())
		})
	}
}

func TestPgBackRestRecoveryConfig_VersionBoundary(t *testing.T) {
	p := newPgBackRest(pgbackrestOptions{Stanza: "mydb"})

	testCases := []struct {
		name           string
		version        float64
		expectTimeline bool
	}{
		{name: "pg 9.6", version: 9.6, expectTimeline: true},
		{name: "pg 11.99", version: 11.99, expectTimeline: true},
		{name: "pg 12.0 exact", version: 12.0, expectTimeline: false},
		{name: "pg 14.0", version: 14.0, expectTimeline: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := p.GetRecoveryConfig(tc.version)
			assert.Contains(t, cfg["restore_command"], "--stanza=mydb")
			if tc.expectTimeline {
				assert.Equal(t, "latest", cfg["recovery_target_timeline"])
			} else {
				_, exists := cfg["recovery_target_timeline"]
				assert.False(t, exists)
			}
		})
	}
}

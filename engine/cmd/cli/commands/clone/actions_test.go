/*
2026 © Postgres.ai
*/

package clone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestUpgradeCommandIsRegistered(t *testing.T) {
	var upgradeCmd *cli.Command

	for _, command := range CommandList() {
		for _, sub := range command.Subcommands {
			if sub.Name == "upgrade" {
				upgradeCmd = sub
			}
		}
	}

	if upgradeCmd == nil {
		t.Fatal("the upgrade subcommand is not registered")
	}

	assert.Equal(t, "CLONE_ID", upgradeCmd.ArgsUsage)
	assert.NotNil(t, upgradeCmd.Before, "the clone ID has to be validated before the action runs")

	flagNames := map[string]bool{}

	for _, f := range upgradeCmd.Flags {
		for _, name := range f.Names() {
			flagNames[name] = true
		}
	}

	assert.True(t, flagNames[cloneUpgradeDockerImageFlag])
	assert.True(t, flagNames["async"])
	assert.False(t, flagNames["target-version"], "the target follows from the instance configuration, not from a flag")
}

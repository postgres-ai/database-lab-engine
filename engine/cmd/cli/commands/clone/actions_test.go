/*
2026 © Postgres.ai
*/

package clone

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
)

// upgradeContext builds a cli.Context carrying the upgrade flags, mirroring how urfave/cli hands
// parsed flags to an action.
func upgradeContext(t *testing.T, args ...string) *cli.Context {
	t.Helper()

	set := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	set.Int(cloneUpgradeTargetVersionFlag, 0, "")
	set.String(cloneUpgradeDockerImageFlag, "", "")

	if err := set.Parse(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	return cli.NewContext(cli.NewApp(), set, nil)
}

func TestBuildUpgradeRequest(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected types.CloneUpgradeRequest
	}{
		{
			name:     "target version only",
			args:     []string{"--target-version", "17"},
			expected: types.CloneUpgradeRequest{TargetVersion: 17},
		},
		{
			name:     "explicit image is passed through",
			args:     []string{"--target-version", "17", "--docker-image", "postgresai/extended-postgres:17-0.8.0"},
			expected: types.CloneUpgradeRequest{TargetVersion: 17, DockerImage: "postgresai/extended-postgres:17-0.8.0"},
		},
		{
			name:     "no flags leaves the request empty for the server to reject",
			args:     nil,
			expected: types.CloneUpgradeRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildUpgradeRequest(upgradeContext(t, tt.args...)))
		})
	}
}

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
	required := map[string]bool{}

	for _, f := range upgradeCmd.Flags {
		for _, name := range f.Names() {
			flagNames[name] = true
		}

		if intFlag, ok := f.(*cli.IntFlag); ok {
			required[intFlag.Name] = intFlag.Required
		}
	}

	assert.True(t, flagNames[cloneUpgradeTargetVersionFlag])
	assert.True(t, flagNames[cloneUpgradeDockerImageFlag])
	assert.True(t, flagNames["async"])
	assert.True(t, required[cloneUpgradeTargetVersionFlag], "an upgrade without a target version is meaningless")
}

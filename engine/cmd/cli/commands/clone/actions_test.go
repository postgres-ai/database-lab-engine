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

func TestSplitFlags(t *testing.T) {
	testCases := []struct {
		name    string
		flags   []string
		want    map[string]string
		wantErr bool
	}{
		{name: "empty", flags: nil, want: map[string]string{}},
		{name: "single pair", flags: []string{"shared_buffers=1GB"}, want: map[string]string{"shared_buffers": "1GB"}},
		{name: "value keeps extra equals", flags: []string{"search_path=a=b"}, want: map[string]string{"search_path": "a=b"}},
		{name: "empty value", flags: []string{"work_mem="}, want: map[string]string{"work_mem": ""}},
		{name: "several pairs", flags: []string{"a=1", "b=2"}, want: map[string]string{"a": "1", "b": "2"}},
		{name: "missing equals", flags: []string{"foo"}, wantErr: true},
		{name: "missing key", flags: []string{"=bar"}, wantErr: true},
		{name: "bad entry among good", flags: []string{"a=1", "foo"}, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := splitFlags(tc.flags)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

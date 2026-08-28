/*
2026 © Postgres.ai
*/

package srv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveUpgradeImage(t *testing.T) {
	const upgradeImage = "postgresai/pg-upgrade:17"

	tests := []struct {
		name     string
		input    upgradeImageInput
		expected string
		wantErr  string
	}{
		{name: "substitutes the major and keeps every suffix",
			input:    upgradeImageInput{CurrentImage: "postgresai/extended-postgres:16-0.8.0-glibc236", CurrentVersion: "16", TargetVersion: 17, PgUpgradeImage: upgradeImage},
			expected: "postgresai/extended-postgres:17-0.8.0-glibc236"},
		{name: "plain major tag",
			input:    upgradeImageInput{CurrentImage: "postgresai/extended-postgres:16", CurrentVersion: "16", TargetVersion: 18, PgUpgradeImage: upgradeImage},
			expected: "postgresai/extended-postgres:18"},
		{name: "SE repository path",
			input:    upgradeImageInput{CurrentImage: "registry.gitlab.com/postgres-ai/se-images/rds:16-0.8.0", CurrentVersion: "16", TargetVersion: 17, PgUpgradeImage: upgradeImage},
			expected: "registry.gitlab.com/postgres-ai/se-images/rds:17-0.8.0"},
		{name: "registry host with a port",
			input:    upgradeImageInput{CurrentImage: "registry:5000/extended-postgres:16-0.8.0", CurrentVersion: "16", TargetVersion: 17, PgUpgradeImage: upgradeImage},
			expected: "registry:5000/extended-postgres:17-0.8.0"},
		{name: "explicit image wins over substitution",
			input:    upgradeImageInput{CurrentImage: "postgresai/extended-postgres:16", CurrentVersion: "16", TargetVersion: 17, RequestedImage: "custom/pg:17-custom", PgUpgradeImage: upgradeImage},
			expected: "custom/pg:17-custom"},
		{name: "explicit image works without a recorded current image",
			input:    upgradeImageInput{TargetVersion: 17, RequestedImage: "custom/pg:17", PgUpgradeImage: upgradeImage},
			expected: "custom/pg:17"},
		{name: "unconfigured feature", input: upgradeImageInput{CurrentImage: "img:16", TargetVersion: 17}, wantErr: "not configured"},
		{name: "missing target version", input: upgradeImageInput{CurrentImage: "img:16", PgUpgradeImage: upgradeImage}, wantErr: "positive PostgreSQL major"},
		{name: "negative target version", input: upgradeImageInput{CurrentImage: "img:16", TargetVersion: -1, PgUpgradeImage: upgradeImage}, wantErr: "positive PostgreSQL major"},
		{name: "downgrade", input: upgradeImageInput{CurrentImage: "img:17", CurrentVersion: "17", TargetVersion: 16, PgUpgradeImage: upgradeImage}, wantErr: "must be greater than"},
		{name: "same version", input: upgradeImageInput{CurrentImage: "img:16", CurrentVersion: "16", TargetVersion: 16, PgUpgradeImage: upgradeImage}, wantErr: "must be greater than"},
		{name: "jump beyond image coverage", input: upgradeImageInput{CurrentImage: "img:13", CurrentVersion: "13", TargetVersion: 18, PgUpgradeImage: upgradeImage}, wantErr: "at most 4 preceding majors"},
		{name: "latest tag cannot be derived", input: upgradeImageInput{CurrentImage: "postgresai/extended-postgres:latest", TargetVersion: 17, PgUpgradeImage: upgradeImage}, wantErr: "is not a release tag"},
		{name: "minor-release tag cannot be derived", input: upgradeImageInput{CurrentImage: "postgres:14.2", TargetVersion: 17, PgUpgradeImage: upgradeImage}, wantErr: "is not a release tag"},
		{name: "pre-release tag is rejected", input: upgradeImageInput{CurrentImage: "postgres:16beta4", TargetVersion: 17, PgUpgradeImage: upgradeImage}, wantErr: "is not a release tag"},
		{name: "branch-named CI tag is rejected", input: upgradeImageInput{CurrentImage: "registry.gitlab.com/postgres-ai/se-images/rds:16-nik-ci", TargetVersion: 17, PgUpgradeImage: upgradeImage}, wantErr: "is not a release tag"},
		{name: "untagged image cannot be derived", input: upgradeImageInput{CurrentImage: "postgresai/extended-postgres", TargetVersion: 17, PgUpgradeImage: upgradeImage}, wantErr: "carries no tag"},
		{name: "no recorded image and no explicit one", input: upgradeImageInput{TargetVersion: 17, PgUpgradeImage: upgradeImage}, wantErr: "no recorded image"},
		{name: "explicit image that is not a reference",
			input:   upgradeImageInput{TargetVersion: 17, RequestedImage: "custom/pg:17; touch /tmp/x", PgUpgradeImage: upgradeImage},
			wantErr: "not a valid image reference"},
		{name: "explicit image from an allowed repository",
			input: upgradeImageInput{TargetVersion: 17, RequestedImage: "postgresai/extended-postgres:17",
				PgUpgradeImage: upgradeImage, AllowedRepositories: []string{"postgresai/extended-postgres"}},
			expected: "postgresai/extended-postgres:17"},
		{name: "explicit image outside the allow list",
			input: upgradeImageInput{TargetVersion: 17, RequestedImage: "evil/pg:17",
				PgUpgradeImage: upgradeImage, AllowedRepositories: []string{"postgresai/extended-postgres"}},
			wantErr: "not listed in provision.upgradeImageAllowList"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, err := resolveUpgradeImage(tt.input)

			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, image)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestValidateRequestedImage(t *testing.T) {
	t.Run("an empty allow list accepts any repository", func(t *testing.T) {
		require.NoError(t, validateRequestedImage("registry.example.com/team/pg:17", nil))
	})

	t.Run("shell syntax never reaches the docker command", func(t *testing.T) {
		for _, image := range []string{"pg:17; rm -rf /", "$(touch /tmp/pwned)", "pg' --privileged '", "pg:17 --volume /:/host"} {
			require.Error(t, validateRequestedImage(image, nil), image)
		}
	})

	t.Run("the allow list matches on the repository, not the tag", func(t *testing.T) {
		allowed := []string{"postgresai/extended-postgres"}

		require.NoError(t, validateRequestedImage("postgresai/extended-postgres:18-0.8.0-glibc236", allowed))
		require.NoError(t, validateRequestedImage("docker.io/postgresai/extended-postgres:18", allowed),
			"the implicit registry is the same repository")
		require.Error(t, validateRequestedImage("postgresai/extended-postgres-fork:18", allowed),
			"a longer name that starts the same is a different repository")
	})

	t.Run("a private registry entry matches only itself", func(t *testing.T) {
		allowed := []string{"registry.example.com/team/pg"}

		require.NoError(t, validateRequestedImage("registry.example.com/team/pg:17", allowed))
		require.Error(t, validateRequestedImage("team/pg:17", allowed))
	})
}

func TestSplitImageTag(t *testing.T) {
	tests := []struct {
		name  string
		image string
		repo  string
		tag   string
	}{
		{name: "simple", image: "postgres:17", repo: "postgres", tag: "17"},
		{name: "namespaced", image: "postgresai/extended-postgres:17-0.8.0", repo: "postgresai/extended-postgres", tag: "17-0.8.0"},
		{name: "registry host with port", image: "registry:5000/repo:17", repo: "registry:5000/repo", tag: "17"},
		{name: "registry host with port and no tag", image: "registry:5000/repo", repo: "registry:5000/repo", tag: ""},
		{name: "no tag", image: "postgres", repo: "postgres", tag: ""},
		{name: "deep path", image: "registry.gitlab.com/postgres-ai/se-images/rds:16-0.8.0", repo: "registry.gitlab.com/postgres-ai/se-images/rds", tag: "16-0.8.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tag := splitImageTag(tt.image)
			assert.Equal(t, tt.repo, repo)
			assert.Equal(t, tt.tag, tag)
		})
	}
}

func TestValidateVersionJump(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  int
		wantErr string
	}{
		{name: "unknown current version defers to the provisioner", current: "", target: 17},
		{name: "unparsable current version defers to the provisioner", current: "unknown", target: 17},
		{name: "legacy dotted current version", current: "9.6", target: 13},
		{name: "one major up", current: "16", target: 17},
		{name: "four majors up", current: "14", target: 18},
		{name: "five majors up", current: "13", target: 18, wantErr: "at most 4 preceding majors"},
		{name: "downgrade", current: "17", target: 16, wantErr: "must be greater than"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVersionJump(tt.current, tt.target)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestCheckDefaultConfigDir(t *testing.T) {
	// The engine ships default configs for majors 10-18; anything outside that has to be
	// rejected before a clone is converted into a cluster the engine cannot configure.
	require.Error(t, checkDefaultConfigDir(99))
}

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
	tests := []struct {
		name     string
		input    upgradeImageInput
		expected string
		wantErr  string
	}{
		{name: "substitutes the major and keeps every suffix",
			input:    upgradeImageInput{CurrentImage: "postgresai/extended-postgres:16-0.8.0-glibc236", CurrentVersion: "16", TargetVersion: 17},
			expected: "postgresai/extended-postgres:17-0.8.0-glibc236"},
		{name: "plain major tag",
			input:    upgradeImageInput{CurrentImage: "postgresai/extended-postgres:16", CurrentVersion: "16", TargetVersion: 18},
			expected: "postgresai/extended-postgres:18"},
		{name: "SE repository path",
			input:    upgradeImageInput{CurrentImage: "registry.gitlab.com/postgres-ai/se-images/rds:16-0.8.0", CurrentVersion: "16", TargetVersion: 17},
			expected: "registry.gitlab.com/postgres-ai/se-images/rds:17-0.8.0"},
		{name: "registry host with a port",
			input:    upgradeImageInput{CurrentImage: "registry:5000/extended-postgres:16-0.8.0", CurrentVersion: "16", TargetVersion: 17},
			expected: "registry:5000/extended-postgres:17-0.8.0"},
		{name: "explicit image wins over substitution",
			input: upgradeImageInput{CurrentImage: "postgresai/extended-postgres:16", CurrentVersion: "16", TargetVersion: 17,
				RequestedImage: "custom/pg:17-custom", AllowedRepositories: []string{"custom/pg"}},
			expected: "custom/pg:17-custom"},
		{name: "explicit image works without a recorded current image",
			input:    upgradeImageInput{TargetVersion: 17, RequestedImage: "custom/pg:17", AllowedRepositories: []string{anyRepository}},
			expected: "custom/pg:17"},
		{name: "explicit image within the default allow list",
			input: upgradeImageInput{CurrentImage: "postgresai/extended-postgres:16", CurrentVersion: "16", TargetVersion: 17,
				RequestedImage:      "postgresai/extended-postgres:17-0.8.0-glibc236",
				AllowedRepositories: allowedUpgradeRepositories(nil, "", "postgresai/extended-postgres:16")},
			expected: "postgresai/extended-postgres:17-0.8.0-glibc236"},
		{name: "explicit image outside the default allow list",
			input: upgradeImageInput{CurrentImage: "postgresai/extended-postgres:16", CurrentVersion: "16", TargetVersion: 17,
				RequestedImage: "evil/pg:17", AllowedRepositories: allowedUpgradeRepositories(nil, "", "postgresai/extended-postgres:16")},
			wantErr: "not listed in provision.upgradeImageAllowList"},
		{name: "downgrade", input: upgradeImageInput{CurrentImage: "img:17", CurrentVersion: "17", TargetVersion: 16}, wantErr: "must be greater than"},
		{name: "same version", input: upgradeImageInput{CurrentImage: "img:16", CurrentVersion: "16", TargetVersion: 16}, wantErr: "must be greater than"},
		{name: "jump beyond image coverage", input: upgradeImageInput{CurrentImage: "img:13", CurrentVersion: "13", TargetVersion: 18}, wantErr: "at most 4 preceding majors"},
		{name: "latest tag cannot be derived", input: upgradeImageInput{CurrentImage: "postgresai/extended-postgres:latest", TargetVersion: 17}, wantErr: "is not a release tag"},
		{name: "minor-release tag cannot be derived", input: upgradeImageInput{CurrentImage: "postgres:14.2", TargetVersion: 17}, wantErr: "is not a release tag"},
		{name: "pre-release tag is rejected", input: upgradeImageInput{CurrentImage: "postgres:16beta4", TargetVersion: 17}, wantErr: "is not a release tag"},
		{name: "branch-named CI tag is rejected", input: upgradeImageInput{CurrentImage: "registry.gitlab.com/postgres-ai/se-images/rds:16-nik-ci", TargetVersion: 17}, wantErr: "is not a release tag"},
		{name: "untagged image cannot be derived", input: upgradeImageInput{CurrentImage: "postgresai/extended-postgres", TargetVersion: 17}, wantErr: "carries no tag"},
		{name: "no recorded image and no explicit one", input: upgradeImageInput{TargetVersion: 17}, wantErr: "no recorded image"},
		{name: "explicit image that is not a reference",
			input:   upgradeImageInput{TargetVersion: 17, RequestedImage: "custom/pg:17; touch /tmp/x"},
			wantErr: "not a valid image reference"},
		{name: "explicit image from an allowed repository",
			input:    upgradeImageInput{TargetVersion: 17, RequestedImage: "postgresai/extended-postgres:17", AllowedRepositories: []string{"postgresai/extended-postgres"}},
			expected: "postgresai/extended-postgres:17"},
		{name: "no resolved target", input: upgradeImageInput{CurrentImage: "img:16"}, wantErr: "positive PostgreSQL major"},
		{name: "explicit image of another major",
			input:   upgradeImageInput{CurrentImage: "postgresai/extended-postgres:16", CurrentVersion: "16", TargetVersion: 17, RequestedImage: "postgresai/extended-postgres:18"},
			wantErr: "but this instance upgrades to 17"},
		{name: "explicit image outside the allow list",
			input:   upgradeImageInput{TargetVersion: 17, RequestedImage: "evil/pg:17", AllowedRepositories: []string{"postgresai/extended-postgres"}},
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
	t.Run("an empty allow list accepts no repository", func(t *testing.T) {
		err := validateRequestedImage("registry.example.com/team/pg:17", 17, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permits no repository")
	})

	t.Run("a wildcard entry accepts any repository", func(t *testing.T) {
		require.NoError(t, validateRequestedImage("registry.example.com/team/pg:17", 17, []string{anyRepository}))
		require.NoError(t, validateRequestedImage("custom/pg:17", 17, []string{"postgresai/extended-postgres", anyRepository}))
	})

	t.Run("shell syntax never reaches the docker command", func(t *testing.T) {
		for _, image := range []string{"pg:17; rm -rf /", "$(touch /tmp/pwned)", "pg' --privileged '", "pg:17 --volume /:/host"} {
			require.Error(t, validateRequestedImage(image, 17, []string{anyRepository}), image)
		}
	})

	t.Run("the allow list matches on the repository, not the tag", func(t *testing.T) {
		allowed := []string{"postgresai/extended-postgres"}

		require.NoError(t, validateRequestedImage("postgresai/extended-postgres:18-0.8.0-glibc236", 18, allowed))
		require.NoError(t, validateRequestedImage("docker.io/postgresai/extended-postgres:18", 18, allowed),
			"the implicit registry is the same repository")
		require.Error(t, validateRequestedImage("postgresai/extended-postgres-fork:18", 18, allowed),
			"a longer name that starts the same is a different repository")
	})

	t.Run("a release tag naming another major is refused", func(t *testing.T) {
		wildcard := []string{anyRepository}

		require.Error(t, validateRequestedImage("postgresai/extended-postgres:18", 17, wildcard))
		require.NoError(t, validateRequestedImage("postgresai/extended-postgres:17-0.8.0-glibc236", 17, wildcard))
		require.NoError(t, validateRequestedImage("custom/pg:17-custom", 17, wildcard),
			"a tag outside the release grammar states no major of its own")
	})

	t.Run("a private registry entry matches only itself", func(t *testing.T) {
		allowed := []string{"registry.example.com/team/pg"}

		require.NoError(t, validateRequestedImage("registry.example.com/team/pg:17", 17, allowed))
		require.Error(t, validateRequestedImage("team/pg:17", 17, allowed))
	})
}

func TestAllowedUpgradeRepositories(t *testing.T) {
	tests := []struct {
		name       string
		configured []string
		images     []string
		expected   []string
	}{
		{name: "configured list wins", configured: []string{"team/pg"}, images: []string{"postgresai/extended-postgres:16"}, expected: []string{"team/pg"}},
		{name: "wildcard is kept verbatim", configured: []string{anyRepository}, images: []string{"postgresai/extended-postgres:16"}, expected: []string{anyRepository}},
		{name: "instance image repository", images: []string{"", "postgresai/extended-postgres:16"}, expected: []string{"docker.io/postgresai/extended-postgres"}},
		{name: "clone override adds its repository",
			images:   []string{"registry.example.com/team/pg:17", "postgresai/extended-postgres:16"},
			expected: []string{"registry.example.com/team/pg", "docker.io/postgresai/extended-postgres"}},
		{name: "same repository is listed once", images: []string{"postgresai/extended-postgres:17", "postgresai/extended-postgres:16"}, expected: []string{"docker.io/postgresai/extended-postgres"}},
		{name: "unparsable image adds nothing", images: []string{"pg:17; rm -rf /", "postgresai/extended-postgres:16"}, expected: []string{"docker.io/postgresai/extended-postgres"}},
		{name: "no image yields no repository", images: []string{"", ""}, expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, allowedUpgradeRepositories(tt.configured, tt.images...))
		})
	}

	t.Run("the default accepts the image the engine derives itself", func(t *testing.T) {
		allowed := allowedUpgradeRepositories(nil, "", "postgresai/extended-postgres:16-0.8.0-glibc236")

		image, err := resolveUpgradeImage(upgradeImageInput{CurrentImage: "postgresai/extended-postgres:16-0.8.0-glibc236",
			CurrentVersion: "16", TargetVersion: 17})
		require.NoError(t, err)
		require.NoError(t, validateRequestedImage(image, 17, allowed))
		require.NoError(t, validateRequestedImage("docker.io/postgresai/extended-postgres:17", 17, allowed))
		require.Error(t, validateRequestedImage("postgresai/pg-upgrade:17", 17, allowed), "the upgrade image is not a clone image")
	})

	t.Run("a private registry clone image admits only its own repository", func(t *testing.T) {
		const repository = "registry.gitlab.com/postgres-ai/custom-images/extended-postgres"

		allowed := allowedUpgradeRepositories(nil, "", repository+":17")

		require.NoError(t, validateRequestedImage(repository+":18", 18, allowed))
		require.NoError(t, validateRequestedImage(repository+":0-does-not-exist", 18, allowed),
			"a tag outside the release grammar states no major; only the repository is checked")
		require.Error(t, validateRequestedImage("postgresai/extended-postgres:18", 18, allowed),
			"the public repository is not the clone image repository")
		require.Error(t, validateRequestedImage("registry.gitlab.com/postgres-ai/database-lab/pg-upgrade:0-does-not-exist", 18, allowed))
	})
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

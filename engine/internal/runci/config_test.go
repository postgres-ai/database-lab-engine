package runci

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/envvar"
)

func TestLoadConfigurationExpandsEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	require.NoError(t, os.Mkdir("configs", 0700))

	t.Setenv("CI_CHECKER_VERIFICATION_TOKEN", "checker-token")
	t.Setenv("DBLAB_VERIFICATION_TOKEN", "dblab-token")
	t.Setenv("PGAI_PLATFORM_ACCESS_TOKEN", "platform-token")
	t.Setenv("VCS_ACCESS_TOKEN", "vcs-token")

	configPath := filepath.Join(tmpDir, "configs", configFilename)
	configData := []byte(`app:
  verificationToken: "${CI_CHECKER_VERIFICATION_TOKEN}"
dle:
  verificationToken: "${DBLAB_VERIFICATION_TOKEN}"
platform:
  url: "https://postgres.ai/api/general"
  accessToken: "${PGAI_PLATFORM_ACCESS_TOKEN}"
source:
  type: "github"
  token: "${VCS_ACCESS_TOKEN}"
runner:
  image: "postgresai/migration-tools:sqitch"
`)
	require.NoError(t, os.WriteFile(configPath, configData, 0600))

	cfg, err := LoadConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "checker-token", cfg.App.VerificationToken)
	assert.Equal(t, "dblab-token", cfg.DLE.VerificationToken)
	assert.Equal(t, "platform-token", cfg.Platform.AccessToken)
	assert.Equal(t, "vcs-token", cfg.Source.Token)
}

func TestLoadConfigurationErrorsOnMissingEnvVariable(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	require.NoError(t, os.Mkdir("configs", 0700))

	configPath := filepath.Join(tmpDir, "configs", configFilename)
	configData := []byte(`app:
  verificationToken: "${RUNCI_MISSING_TOKEN}"
`)
	require.NoError(t, os.WriteFile(configPath, configData, 0600))

	_, err := LoadConfiguration()
	require.Error(t, err)
	assert.True(t, errors.Is(err, envvar.ErrUnsetEnv))
	assert.Contains(t, err.Error(), "RUNCI_MISSING_TOKEN")
	assert.Contains(t, err.Error(), "line 2")
}

func TestLoadConfigurationExpandsEveryField(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	require.NoError(t, os.Mkdir("configs", 0700))

	t.Setenv("CI_CHECKER_VERIFICATION_TOKEN", "checker-token")
	t.Setenv("RUNNER_IMAGE", "registry.example.com/migration-tools:v1.2.3")

	configPath := filepath.Join(tmpDir, "configs", configFilename)
	configData := []byte(`app:
  verificationToken: "${CI_CHECKER_VERIFICATION_TOKEN}"
  port: ${RUNCI_PORT}
runner:
  image: "${RUNNER_IMAGE}"
`)
	require.NoError(t, os.WriteFile(configPath, configData, 0600))

	t.Setenv("RUNCI_PORT", "2500")

	cfg, err := LoadConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "registry.example.com/migration-tools:v1.2.3", cfg.Runner.Image)
	assert.Equal(t, "checker-token", cfg.App.VerificationToken)
	assert.Equal(t, uint(2500), cfg.App.Port)
}

func TestLoadConfigurationKeepsLiteralDollar(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	require.NoError(t, os.Mkdir("configs", 0700))

	t.Setenv("VERSION", "v1.2.3")

	configPath := filepath.Join(tmpDir, "configs", configFilename)
	configData := []byte(`app:
  verificationToken: "p@$$w0rd"
runner:
  image: "registry.example.com/migration-tools:$VERSION"
`)
	require.NoError(t, os.WriteFile(configPath, configData, 0600))

	cfg, err := LoadConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "p@$$w0rd", cfg.App.VerificationToken, "a secret is never reinterpreted")
	assert.Equal(t, "registry.example.com/migration-tools:$VERSION", cfg.Runner.Image,
		"a placeholder inside a longer value is a literal")
}

func TestShippedExampleConfigLoads(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	example := filepath.Join(cwd, "..", "..", "configs", "config.example.ci_checker.yml")

	data, err := os.ReadFile(example)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	require.NoError(t, os.Mkdir("configs", 0700))

	for _, name := range []string{
		"CI_CHECKER_VERIFICATION_TOKEN",
		"DBLAB_VERIFICATION_TOKEN",
		"PGAI_PLATFORM_ACCESS_TOKEN",
		"VCS_ACCESS_TOKEN",
	} {
		t.Setenv(name, "resolved-"+name)
	}

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "configs", configFilename), data, 0600))

	cfg, err := LoadConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "resolved-CI_CHECKER_VERIFICATION_TOKEN", cfg.App.VerificationToken)
	assert.Equal(t, "resolved-VCS_ACCESS_TOKEN", cfg.Source.Token)
}

package runci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Contains(t, err.Error(), "app.verificationToken")
	assert.Contains(t, err.Error(), `"RUNCI_MISSING_TOKEN" is not set`)
}

func TestLoadConfigurationPreservesDollarSignsOutsideTokenFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	require.NoError(t, os.Mkdir("configs", 0700))

	t.Setenv("CI_CHECKER_VERIFICATION_TOKEN", "checker-token")

	configPath := filepath.Join(tmpDir, "configs", configFilename)
	configData := []byte(`app:
  verificationToken: "${CI_CHECKER_VERIFICATION_TOKEN}"
runner:
  image: "registry.example.com/migration-tools:$VERSION"
`)
	require.NoError(t, os.WriteFile(configPath, configData, 0600))

	cfg, err := LoadConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "registry.example.com/migration-tools:$VERSION", cfg.Runner.Image)
}

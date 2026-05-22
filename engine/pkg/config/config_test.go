package config

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/suite"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

func TestLoadConfig(t *testing.T) {
	suite.Run(t, &ConfigSuite{})
}

func copyFile(src, dst string, process func([]byte) []byte) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, process(data), 0600)
}

type ConfigSuite struct {
	suite.Suite
	mountDir string
}

func (s *ConfigSuite) SetupTest() {
	t := s.T()

	s.mountDir = t.TempDir()
	t.Log(s.mountDir)

	cwd, err := os.Getwd()
	s.Require().NoError(err)

	t.Chdir(s.mountDir)

	s.Require().NoError(os.Mkdir("configs", 0700))
	s.Require().NoError(os.Mkdir("data", 0700))

	exampleSrc := filepath.Join(cwd, "../../configs/config.example.logical_generic.yml")
	testConfig := filepath.Join(s.mountDir, "configs/server.yml")

	s.Require().NoError(copyFile(exampleSrc, testConfig, func(data []byte) []byte {
		return bytes.ReplaceAll(data, []byte("/var/lib/dblab"), []byte(s.mountDir))
	}))
}

func (s *ConfigSuite) TestGenerateNewID() {
	instanceID, err := LoadInstanceID()
	s.Require().NoError(err)
	s.NotEmpty(instanceID)

	instanceIDPath, err := util.GetMetaPath("instance_id")
	s.Require().NoError(err)
	data, err := os.ReadFile(instanceIDPath)
	s.Require().NoError(err)
	s.Equal(instanceID, string(data))
}

func (s *ConfigSuite) TestLoadInstanceID() {
	expected := xid.New().String()

	instanceIDPath, err := util.GetMetaPath("instance_id")
	s.Require().NoError(err)
	err = os.MkdirAll(path.Dir(instanceIDPath), 0755)
	s.Require().NoError(err)
	err = os.WriteFile(instanceIDPath, []byte(expected), 0600)
	s.Require().NoError(err)

	loaded, err := LoadInstanceID()
	s.Require().NoError(err)
	s.Equal(expected, loaded)
}

func (s *ConfigSuite) TestLoadInstanceIDMissingFile() {
	loaded, err := LoadInstanceID()
	s.Require().NoError(err)
	s.NotEmpty(loaded)

	instanceIDPath, err := util.GetMetaPath("instance_id")
	s.Require().NoError(err)
	data, err := os.ReadFile(instanceIDPath)
	s.Require().NoError(err)
	s.Equal(loaded, string(data))
}

func (s *ConfigSuite) TestLoadInstanceIDEmptyFile() {
	instanceIDPath, err := util.GetMetaPath("instance_id")
	s.Require().NoError(err)
	s.Require().NoError(os.MkdirAll(path.Dir(instanceIDPath), 0755))
	s.Require().NoError(os.WriteFile(instanceIDPath, []byte{}, 0600))

	loaded, err := LoadInstanceID()
	s.Require().NoError(err)
	s.Empty(loaded)
}

func (s *ConfigSuite) TestGetConfigBytes() {
	b, err := GetConfigBytes()
	s.Require().NoError(err)
	s.NotEmpty(b)
	s.Contains(string(b), "retrieval")
}

func (s *ConfigSuite) TestLoadConfigurationExpandsEnvironmentVariables() {
	t := s.T()

	t.Setenv("DBLAB_VERIFICATION_TOKEN", "env-verification-token")
	t.Setenv("PGAI_PLATFORM_ACCESS_TOKEN", "env-platform-token")
	t.Setenv("DBLAB_WEBHOOK_SECRET", "env-webhook-secret")

	configPath, err := util.GetConfigPath("server.yml")
	s.Require().NoError(err)

	configData := []byte(`server:
  verificationToken: "${DBLAB_VERIFICATION_TOKEN}"
platform:
  url: "https://postgres.ai/api/general"
  accessToken: "${PGAI_PLATFORM_ACCESS_TOKEN}"
webhooks:
  hooks:
    - url: "https://example.com/hook"
      secret: "${DBLAB_WEBHOOK_SECRET}"
      trigger: ["clone_create"]
`)
	s.Require().NoError(os.WriteFile(configPath, configData, 0600))

	cfg, err := LoadConfiguration()
	s.Require().NoError(err)
	s.Equal("env-verification-token", cfg.Server.VerificationToken)
	s.Equal("env-platform-token", cfg.Platform.AccessToken)
	s.Require().Len(cfg.Webhooks.Hooks, 1)
	s.Equal("env-webhook-secret", cfg.Webhooks.Hooks[0].Secret)
}

func (s *ConfigSuite) TestLoadConfigurationErrorsOnMissingEnvVariable() {
	configPath, err := util.GetConfigPath("server.yml")
	s.Require().NoError(err)

	configData := []byte(`server:
  verificationToken: "${DBLAB_MISSING_TOKEN}"
`)
	s.Require().NoError(os.WriteFile(configPath, configData, 0600))

	_, err = LoadConfiguration()
	s.Require().Error(err)
	s.Contains(err.Error(), "server.verificationToken")
	s.Contains(err.Error(), `"DBLAB_MISSING_TOKEN" is not set`)
}

func (s *ConfigSuite) TestLoadConfigurationPreservesDollarSignsOutsideTokenFields() {
	t := s.T()

	t.Setenv("DBLAB_VERIFICATION_TOKEN", "env-verification-token")

	configPath, err := util.GetConfigPath("server.yml")
	s.Require().NoError(err)

	configData := []byte(`server:
  verificationToken: "${DBLAB_VERIFICATION_TOKEN}"
observer:
  replacementRules:
    "[a-z0-9._%+\\-]+(@[a-z0-9.\\-]+\\.[a-z]{2,4})": "***$1"
    "select \\d+": "***"
`)
	s.Require().NoError(os.WriteFile(configPath, configData, 0600))

	cfg, err := LoadConfiguration()
	s.Require().NoError(err)
	s.Equal("env-verification-token", cfg.Server.VerificationToken)
	s.Equal("***$1", cfg.Observer.ReplacementRules[`[a-z0-9._%+\-]+(@[a-z0-9.\-]+\.[a-z]{2,4})`])
}

func (s *ConfigSuite) TestRotateConfig() {
	original, err := GetConfigBytes()
	s.Require().NoError(err)

	newContent := []byte("server:\n  port: 9999\n")
	err = RotateConfig(newContent)
	s.Require().NoError(err)

	updated, err := GetConfigBytes()
	s.Require().NoError(err)
	s.Equal(newContent, updated)

	configPath, err := util.GetConfigPath("server.yml")
	s.Require().NoError(err)

	matches, err := filepath.Glob(configPath + "*.bak")
	s.Require().NoError(err)
	s.NotEmpty(matches, "backup file should be created")

	backupData, err := os.ReadFile(matches[0])
	s.Require().NoError(err)
	s.Equal(original, backupData)
}

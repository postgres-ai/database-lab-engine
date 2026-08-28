package config

import (
	"bytes"
	"errors"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/envvar"
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
	// exampleDir holds the repository's configs directory, captured before the
	// working directory moves to the temporary mount.
	exampleDir string
}

func (s *ConfigSuite) SetupTest() {
	t := s.T()

	s.mountDir = t.TempDir()
	t.Log(s.mountDir)

	cwd, err := os.Getwd()
	s.Require().NoError(err)

	s.exampleDir = filepath.Join(cwd, "..", "..", "configs")

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
	s.True(errors.Is(err, envvar.ErrUnsetEnv))
	s.Contains(err.Error(), "DBLAB_MISSING_TOKEN")
	// The position replaces the field path: expansion runs on the document, so
	// it can point at any scalar, including ones no struct field names.
	s.Contains(err.Error(), "line 2")
}

func (s *ConfigSuite) TestLoadConfigurationPreservesDollarSignsOutsideTokenFields() {
	t := s.T()

	t.Setenv("DBLAB_VERIFICATION_TOKEN", "env-verification-token")

	configPath, err := util.GetConfigPath("server.yml")
	s.Require().NoError(err)

	configData := []byte(`server:
  verificationToken: "${DBLAB_VERIFICATION_TOKEN}"
platform:
  accessToken: "p@$$w0rd"
observer:
  replacementRules:
    "[a-z0-9._%+\\-]+(@[a-z0-9.\\-]+\\.[a-z]{2,4})": "***$1"
    "select \\d+": "***"
`)
	s.Require().NoError(os.WriteFile(configPath, configData, 0600))

	cfg, err := LoadConfiguration()
	s.Require().NoError(err)
	s.Equal("env-verification-token", cfg.Server.VerificationToken)
	s.Equal("p@$$w0rd", cfg.Platform.AccessToken, "a secret is never reinterpreted")
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

func (s *ConfigSuite) TestLoadConfigurationExpandsRetrievalJobOptions() {
	t := s.T()

	t.Setenv("DBLAB_VERIFICATION_TOKEN", "env-verification-token")
	t.Setenv("SOURCE_DB_HOST", "db.example.com")
	t.Setenv("SOURCE_DB_PASSWORD", "s3cr3t-p@ssword")

	configPath, err := util.GetConfigPath("server.yml")
	s.Require().NoError(err)

	configData := []byte(`server:
  verificationToken: "${DBLAB_VERIFICATION_TOKEN}"
retrieval:
  jobs:
    - logicalDump
  spec:
    logicalDump:
      options:
        source:
          type: remote
          connection:
            dbname: postgres
            host: "${SOURCE_DB_HOST}"
            port: 5432
            username: postgres
            password: "${SOURCE_DB_PASSWORD}"
`)
	s.Require().NoError(os.WriteFile(configPath, configData, 0600))

	cfg, err := LoadConfiguration()
	s.Require().NoError(err)

	spec, ok := cfg.Retrieval.JobsSpec["logicalDump"]
	s.Require().True(ok)

	// Nested option values stay map[interface{}]interface{}: the document is
	// walked with yaml.v3 but decoded with yaml.v2, so what the retrieval jobs
	// receive is shaped exactly as before.
	source, ok := spec.Options["source"].(map[interface{}]interface{})
	s.Require().True(ok)

	connection, ok := source["connection"].(map[interface{}]interface{})
	s.Require().True(ok)

	s.Equal("db.example.com", connection["host"])
	s.Equal("s3cr3t-p@ssword", connection["password"])
}

func (s *ConfigSuite) TestLoadConfigurationKeepsAmbiguousSecretsIntact() {
	t := s.T()

	configPath, err := util.GetConfigPath("server.yml")
	s.Require().NoError(err)

	for _, password := range []string{"no", "yes", "off", "0755", "007", "1e5", "1_000", "null", "~", "3m", "1:30"} {
		s.Run(password, func() {
			t.Setenv("SOURCE_DB_PASSWORD", password)

			configData := []byte(`retrieval:
  jobs:
    - logicalDump
  spec:
    logicalDump:
      options:
        source:
          connection:
            password: ${SOURCE_DB_PASSWORD}
`)
			s.Require().NoError(os.WriteFile(configPath, configData, 0600))

			cfg, err := LoadConfiguration()
			s.Require().NoError(err)

			source, ok := cfg.Retrieval.JobsSpec["logicalDump"].Options["source"].(map[interface{}]interface{})
			s.Require().True(ok)

			connection, ok := source["connection"].(map[interface{}]interface{})
			s.Require().True(ok)
			s.Equal(password, connection["password"])
		})
	}
}

func (s *ConfigSuite) TestLoadConfigurationExpandsTypedFields() {
	t := s.T()

	t.Setenv("DBLAB_PORT", "3000")
	t.Setenv("DBLAB_DEBUG", "true")

	configPath, err := util.GetConfigPath("server.yml")
	s.Require().NoError(err)

	configData := []byte(`server:
  port: ${DBLAB_PORT}
global:
  debug: ${DBLAB_DEBUG}
`)
	s.Require().NoError(os.WriteFile(configPath, configData, 0600))

	cfg, err := LoadConfiguration()
	s.Require().NoError(err)
	s.Equal(uint(3000), cfg.Server.Port)
	s.True(cfg.Global.Debug)
}

func (s *ConfigSuite) TestGetConfigBytesKeepsPlaceholders() {
	t := s.T()

	t.Setenv("DBLAB_VERIFICATION_TOKEN", "env-verification-token")

	configPath, err := util.GetConfigPath("server.yml")
	s.Require().NoError(err)

	configData := []byte(`server:
  verificationToken: "${DBLAB_VERIFICATION_TOKEN}"
`)
	s.Require().NoError(os.WriteFile(configPath, configData, 0600))

	cfg, err := LoadConfiguration()
	s.Require().NoError(err)
	s.Equal("env-verification-token", cfg.Server.VerificationToken)

	raw, err := GetConfigBytes()
	s.Require().NoError(err)
	s.Contains(string(raw), "${DBLAB_VERIFICATION_TOKEN}")
	s.NotContains(string(raw), "env-verification-token")

	// The admin API round-trip goes through a yaml.Node; the placeholder has to
	// survive that too, since the result is what RotateConfig persists.
	var node yaml.Node
	s.Require().NoError(yaml.Unmarshal(raw, &node))

	rewritten, err := yaml.Marshal(&node)
	s.Require().NoError(err)
	s.Contains(string(rewritten), "${DBLAB_VERIFICATION_TOKEN}")
	s.NotContains(string(rewritten), "env-verification-token")
}

func (s *ConfigSuite) TestShippedExampleConfigsLoad() {
	t := s.T()

	for _, name := range []string{
		"DBLAB_VERIFICATION_TOKEN",
		"PGAI_PLATFORM_ACCESS_TOKEN",
	} {
		t.Setenv(name, "resolved-"+name)
	}

	examples, err := filepath.Glob(filepath.Join(s.exampleDir, "config.example.*.yml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(examples)

	configPath, err := util.GetConfigPath("server.yml")
	s.Require().NoError(err)

	for _, example := range examples {
		if filepath.Base(example) == "config.example.ci_checker.yml" {
			continue // Migration checker config, loaded by internal/runci.
		}

		s.Run(filepath.Base(example), func() {
			s.Require().NoError(copyFile(example, configPath, func(data []byte) []byte {
				return bytes.ReplaceAll(data, []byte("/var/lib/dblab"), []byte(s.mountDir))
			}))

			cfg, err := LoadConfiguration()
			s.Require().NoError(err)
			s.Equal("resolved-DBLAB_VERIFICATION_TOKEN", cfg.Server.VerificationToken)
		})
	}
}

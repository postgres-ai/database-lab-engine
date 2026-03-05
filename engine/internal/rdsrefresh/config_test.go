/*
2025 © PostgresAI
*/

package rdsrefresh

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")

		cfgContent := `
source:
  type: rds
  identifier: my-prod-db
  dbName: postgres
  username: postgres
  password: secret123
clone:
  instanceClass: db.t3.medium
  securityGroups: [sg-123]
dblab:
  apiEndpoint: https://dblab:2345
  token: test-token
aws:
  region: us-east-1
`
		require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0600))

		cfg, err := LoadConfig(cfgPath)
		require.NoError(t, err)
		assert.Equal(t, "rds", cfg.Source.Type)
		assert.Equal(t, "my-prod-db", cfg.Source.Identifier)
		assert.Equal(t, "postgres", cfg.Source.DBName)
		assert.Equal(t, "postgres", cfg.Source.Username)
		assert.Equal(t, "secret123", cfg.Source.Password)
		assert.Equal(t, "db.t3.medium", cfg.RDSClone.InstanceClass)
		assert.Equal(t, []string{"sg-123"}, cfg.RDSClone.VPCSecurityGroupIDs)
		assert.Equal(t, "https://dblab:2345", cfg.DBLab.APIEndpoint)
		assert.Equal(t, "test-token", cfg.DBLab.Token)
		assert.Equal(t, "us-east-1", cfg.AWS.Region)
	})

	t.Run("config with aurora-cluster", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")

		cfgContent := `
source:
  type: aurora-cluster
  identifier: my-aurora-cluster
  dbName: postgres
  username: postgres
  password: secret123
clone:
  instanceClass: db.r5.large
dblab:
  apiEndpoint: https://dblab:2345
  token: test-token
aws:
  region: eu-west-1
`
		require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0600))

		cfg, err := LoadConfig(cfgPath)
		require.NoError(t, err)
		assert.Equal(t, "aurora-cluster", cfg.Source.Type)
		assert.Equal(t, "my-aurora-cluster", cfg.Source.Identifier)
	})

	t.Run("config with environment variables", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")

		t.Setenv("DB_PASSWORD", "secret-from-env")
		t.Setenv("DBLAB_TOKEN", "token-from-env")

		cfgContent := `
source:
  type: rds
  identifier: my-prod-db
  dbName: postgres
  username: postgres
  password: ${DB_PASSWORD}
clone:
  instanceClass: db.t3.medium
dblab:
  apiEndpoint: https://dblab:2345
  token: ${DBLAB_TOKEN}
aws:
  region: us-east-1
`
		require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0600))

		cfg, err := LoadConfig(cfgPath)
		require.NoError(t, err)
		assert.Equal(t, "secret-from-env", cfg.Source.Password)
		assert.Equal(t, "token-from-env", cfg.DBLab.Token)
	})

	t.Run("config with custom durations", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")

		cfgContent := `
source:
  type: rds
  identifier: my-prod-db
  dbName: postgres
  username: postgres
  password: secret123
clone:
  instanceClass: db.t3.medium
dblab:
  apiEndpoint: https://dblab:2345
  token: test-token
  pollInterval: 1m
  timeout: 2h
aws:
  region: us-east-1
`
		require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0600))

		cfg, err := LoadConfig(cfgPath)
		require.NoError(t, err)
		assert.Equal(t, time.Minute, cfg.DBLab.PollInterval.Duration())
		assert.Equal(t, 2*time.Hour, cfg.DBLab.Timeout.Duration())
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/config.yaml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")

		require.NoError(t, os.WriteFile(cfgPath, []byte("invalid: yaml: content:"), 0600))

		_, err := LoadConfig(cfgPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})
}

func TestConfigValidate(t *testing.T) {
	baseConfig := func() *Config {
		return &Config{
			Source:   SourceConfig{Type: "rds", Identifier: "test-db", DBName: "postgres", Username: "postgres", Password: "secret"},
			RDSClone: RDSCloneConfig{InstanceClass: "db.t3.medium"},
			DBLab:    DBLabConfig{APIEndpoint: "https://dblab:2345", Token: "test-token"},
			AWS:      AWSConfig{Region: "us-east-1"},
		}
	}

	t.Run("valid rds config", func(t *testing.T) {
		cfg := baseConfig()
		require.NoError(t, cfg.Validate())
	})

	t.Run("valid aurora-cluster config", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Source.Type = "aurora-cluster"
		require.NoError(t, cfg.Validate())
	})

	t.Run("missing source type", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Source.Type = ""
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source.type is required")
	})

	t.Run("invalid source type", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Source.Type = "invalid-type"
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source.type must be")
		assert.Contains(t, err.Error(), "invalid-type")
	})

	t.Run("missing source identifier", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Source.Identifier = ""
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source.identifier is required")
	})

	t.Run("missing source dbName", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Source.DBName = ""
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source.dbName is required")
	})

	t.Run("missing source username", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Source.Username = ""
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source.username is required")
	})

	t.Run("missing source password", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Source.Password = ""
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source.password is required")
	})

	t.Run("missing clone instanceClass", func(t *testing.T) {
		cfg := baseConfig()
		cfg.RDSClone.InstanceClass = ""
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "clone.instanceClass is required")
	})

	t.Run("missing dblab apiEndpoint", func(t *testing.T) {
		cfg := baseConfig()
		cfg.DBLab.APIEndpoint = ""
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dblab.apiEndpoint is required")
	})

	t.Run("missing dblab token", func(t *testing.T) {
		cfg := baseConfig()
		cfg.DBLab.Token = ""
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dblab.token is required")
	})

	t.Run("missing aws region", func(t *testing.T) {
		cfg := baseConfig()
		cfg.AWS.Region = ""
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "aws.region is required")
	})
}

func TestConfigSetDefaults(t *testing.T) {
	t.Run("sets default poll interval", func(t *testing.T) {
		cfg := &Config{}
		cfg.SetDefaults()
		assert.Equal(t, defaultPollInterval, cfg.DBLab.PollInterval.Duration())
	})

	t.Run("sets default timeout", func(t *testing.T) {
		cfg := &Config{}
		cfg.SetDefaults()
		assert.Equal(t, defaultTimeout, cfg.DBLab.Timeout.Duration())
	})

	t.Run("preserves custom poll interval", func(t *testing.T) {
		cfg := &Config{DBLab: DBLabConfig{PollInterval: Duration(time.Minute)}}
		cfg.SetDefaults()
		assert.Equal(t, time.Minute, cfg.DBLab.PollInterval.Duration())
	})

	t.Run("preserves custom timeout", func(t *testing.T) {
		cfg := &Config{DBLab: DBLabConfig{Timeout: Duration(time.Hour)}}
		cfg.SetDefaults()
		assert.Equal(t, time.Hour, cfg.DBLab.Timeout.Duration())
	})

	t.Run("adds managed-by tags", func(t *testing.T) {
		cfg := &Config{}
		cfg.SetDefaults()
		assert.NotNil(t, cfg.RDSClone.Tags)
		assert.Equal(t, "dblab-rds-refresh", cfg.RDSClone.Tags["ManagedBy"])
		assert.Equal(t, "true", cfg.RDSClone.Tags["AutoDelete"])
	})

	t.Run("preserves existing tags", func(t *testing.T) {
		cfg := &Config{RDSClone: RDSCloneConfig{Tags: map[string]string{"Custom": "value"}}}
		cfg.SetDefaults()
		assert.Equal(t, "value", cfg.RDSClone.Tags["Custom"])
		assert.Equal(t, "dblab-rds-refresh", cfg.RDSClone.Tags["ManagedBy"])
		assert.Equal(t, "true", cfg.RDSClone.Tags["AutoDelete"])
	})
}

func TestDurationMarshaling(t *testing.T) {
	t.Run("marshal duration", func(t *testing.T) {
		d := Duration(30 * time.Second)
		result, err := d.MarshalYAML()
		require.NoError(t, err)
		assert.Equal(t, "30s", result)
	})

	t.Run("duration method", func(t *testing.T) {
		d := Duration(2 * time.Hour)
		assert.Equal(t, 2*time.Hour, d.Duration())
	})
}

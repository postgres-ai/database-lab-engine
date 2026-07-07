package models

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/projection"
)

// loadExampleYaml loads and unmarshals an engine/configs/config.example.*.yml file.
func loadExampleYaml(t *testing.T, name string) *yaml.Node {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	path := filepath.Join(cwd, "..", "..", "configs", name)

	data, err := os.ReadFile(path)
	require.NoError(t, err, "read %s", path)

	node := &yaml.Node{}
	require.NoError(t, yaml.Unmarshal(data, node))

	return node
}

// loadTestdataYaml loads and unmarshals a yml fixture from this package's testdata/.
func loadTestdataYaml(t *testing.T, name string) *yaml.Node {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	path := filepath.Join(cwd, "testdata", name)

	data, err := os.ReadFile(path)
	require.NoError(t, err, "read %s", path)

	node := &yaml.Node{}
	require.NoError(t, yaml.Unmarshal(data, node))

	return node
}

// loadProjection loads the projection from a yaml document for both default and sensitive groups.
func loadProjection(t *testing.T, node *yaml.Node) *ConfigProjection {
	t.Helper()

	proj := &ConfigProjection{}
	require.NoError(t, projection.LoadYaml(proj, node, projection.LoadOptions{
		Groups: []string{"default", "sensitive"},
	}))

	return proj
}

func TestConfigProjection_PhysicalWalg(t *testing.T) {
	node := loadExampleYaml(t, "config.example.physical_walg.yml")
	proj := loadProjection(t, node)

	require.NotNil(t, proj.PhysicalTool)
	require.Equal(t, "walg", *proj.PhysicalTool)

	require.NotNil(t, proj.PhysicalWalgBackupName)
	require.Equal(t, "LATEST", *proj.PhysicalWalgBackupName)

	require.NotNil(t, proj.PhysicalSyncEnabled)
	require.True(t, *proj.PhysicalSyncEnabled)

	require.NotEmpty(t, proj.PhysicalEnvs)
	require.Contains(t, proj.PhysicalEnvs, "WALG_GS_PREFIX")
	require.Contains(t, proj.PhysicalEnvs, "GOOGLE_APPLICATION_CREDENTIALS")

	require.Nil(t, proj.PhysicalPgbackrestStanza)
	require.Nil(t, proj.PhysicalPgbackrestDelta)

	require.Nil(t, proj.Host, "logical-mode source fields must stay nil for physical config")
	require.Nil(t, proj.Username)
}

func TestConfigProjection_PhysicalPgbackrest(t *testing.T) {
	node := loadExampleYaml(t, "config.example.physical_pgbackrest.yml")
	proj := loadProjection(t, node)

	require.NotNil(t, proj.PhysicalTool)
	require.Equal(t, "pgbackrest", *proj.PhysicalTool)

	require.NotNil(t, proj.PhysicalPgbackrestStanza)
	require.Equal(t, "stanzaName", *proj.PhysicalPgbackrestStanza)

	require.NotNil(t, proj.PhysicalPgbackrestDelta)
	require.False(t, *proj.PhysicalPgbackrestDelta)

	require.NotEmpty(t, proj.PhysicalEnvs)
	require.Contains(t, proj.PhysicalEnvs, "PGBACKREST_REPO")

	require.Nil(t, proj.PhysicalWalgBackupName)
}

func TestConfigProjection_PhysicalGeneric_CustomTool(t *testing.T) {
	node := loadExampleYaml(t, "config.example.physical_generic.yml")
	proj := loadProjection(t, node)

	require.NotNil(t, proj.PhysicalTool, "customTool example must still populate the tool field")
	require.Equal(t, "customTool", *proj.PhysicalTool)

	require.Nil(t, proj.PhysicalWalgBackupName, "walg fields must stay nil for customTool")
	require.Nil(t, proj.PhysicalPgbackrestStanza, "pgbackrest fields must stay nil for customTool")
	require.Nil(t, proj.PhysicalPgbackrestDelta)
}

func TestConfigProjection_LogicalGeneric(t *testing.T) {
	node := loadExampleYaml(t, "config.example.logical_generic.yml")
	proj := loadProjection(t, node)

	require.NotNil(t, proj.DBName)
	require.Equal(t, "postgres", *proj.DBName)
	require.NotNil(t, proj.Username)
	require.Equal(t, "postgres", *proj.Username)

	require.Nil(t, proj.PhysicalTool, "physical fields must stay nil for logical config")
	require.Nil(t, proj.PhysicalWalgBackupName)
	require.Nil(t, proj.PhysicalPgbackrestStanza)
	require.Empty(t, proj.PhysicalEnvs)
}

func TestConfigProjection_RoundTrip_PhysicalWalg(t *testing.T) {
	node := loadExampleYaml(t, "config.example.physical_walg.yml")
	first := loadProjection(t, node)

	require.NoError(t, projection.StoreYaml(first, node, projection.StoreOptions{
		Groups: []string{"default", "sensitive"},
	}))

	second := loadProjection(t, node)

	require.Equal(t, first.PhysicalTool, second.PhysicalTool)
	require.Equal(t, first.PhysicalWalgBackupName, second.PhysicalWalgBackupName)
	require.Equal(t, first.PhysicalSyncEnabled, second.PhysicalSyncEnabled)
	require.Equal(t, first.PhysicalEnvs, second.PhysicalEnvs)
}

func TestConfigProjection_RoundTrip_PhysicalPgbackrest(t *testing.T) {
	node := loadExampleYaml(t, "config.example.physical_pgbackrest.yml")
	first := loadProjection(t, node)

	require.NoError(t, projection.StoreYaml(first, node, projection.StoreOptions{
		Groups: []string{"default", "sensitive"},
	}))

	second := loadProjection(t, node)

	require.Equal(t, first.PhysicalTool, second.PhysicalTool)
	require.Equal(t, first.PhysicalPgbackrestStanza, second.PhysicalPgbackrestStanza)
	require.Equal(t, first.PhysicalPgbackrestDelta, second.PhysicalPgbackrestDelta)
	require.Equal(t, first.PhysicalEnvs, second.PhysicalEnvs)
}

func TestConfigProjection_PhysicalEnvs_CreateKey(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		fromTestdata bool
		setEnvs      map[string]interface{}
		mustContain  map[string]interface{}
	}{
		{name: "no envs key — writer creates it", fixture: "physical_no_envs.yml", fromTestdata: true, setEnvs: map[string]interface{}{"AWS_REGION": "us-east-1"}, mustContain: map[string]interface{}{"AWS_REGION": "us-east-1"}},
		{name: "existing envs key — writer overwrites with new map", fixture: "config.example.physical_walg.yml", fromTestdata: false, setEnvs: map[string]interface{}{"WALG_GS_PREFIX": "gs://existing/path", "GOOGLE_APPLICATION_CREDENTIALS": "/tmp/sa.json", "NEW_KEY": "v"}, mustContain: map[string]interface{}{"NEW_KEY": "v", "WALG_GS_PREFIX": "gs://existing/path"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var node *yaml.Node
			if tc.fromTestdata {
				node = loadTestdataYaml(t, tc.fixture)
			} else {
				node = loadExampleYaml(t, tc.fixture)
			}

			proj := &ConfigProjection{PhysicalEnvs: tc.setEnvs}
			require.NoError(t, projection.StoreYaml(proj, node, projection.StoreOptions{
				Groups: []string{"default", "sensitive"},
			}))

			out, err := yaml.Marshal(node)
			require.NoError(t, err, "marshal updated yaml")
			require.Contains(t, string(out), "envs:", "envs key must be present after write")

			reloaded := loadProjection(t, node)
			require.NotEmpty(t, reloaded.PhysicalEnvs, "envs map must be readable after write")
			for k, v := range tc.mustContain {
				require.Contains(t, reloaded.PhysicalEnvs, k)
				require.Equal(t, v, reloaded.PhysicalEnvs[k])
			}
		})
	}
}

func TestConfigProjection_RoundTrip_LogicalGeneric(t *testing.T) {
	node := loadExampleYaml(t, "config.example.logical_generic.yml")
	first := loadProjection(t, node)

	require.NoError(t, projection.StoreYaml(first, node, projection.StoreOptions{
		Groups: []string{"default", "sensitive"},
	}))

	second := loadProjection(t, node)

	require.Equal(t, first.Host, second.Host)
	require.Equal(t, first.Username, second.Username)
	require.Equal(t, first.DBName, second.DBName)
	require.Equal(t, first.PhysicalTool, second.PhysicalTool)
}

package config

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
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
	oldCwd   string
	mountDir string
}

func (s *ConfigSuite) SetupTest() {
	t := s.T()

	s.mountDir = t.TempDir()

	t.Log(s.mountDir)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	s.oldCwd = cwd
	require.NoError(t, os.Chdir(s.mountDir))

	require.NoError(t, os.Mkdir("configs", 0700))
	require.NoError(t, os.Mkdir("data", 0700))

	exampleSrc := filepath.Join(cwd, "../../configs/config.example.logical_generic.yml")
	testConfig := filepath.Join(s.mountDir, "configs/server.yml")

	err = copyFile(exampleSrc, testConfig, func(data []byte) []byte {
		return bytes.ReplaceAll(data, []byte("/var/lib/dblab"), []byte(s.mountDir))
	})
	require.NoError(t, err)
}

func (s *ConfigSuite) TearDownTest() {
	s.Require().NoError(os.Chdir(s.oldCwd))
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
	instanceID := xid.New().String()

	instanceIDPath, err := util.GetMetaPath("instance_id")
	s.Require().NoError(err)
	err = os.MkdirAll(path.Dir(instanceIDPath), 0755)
	s.Require().NoError(err)
	err = os.WriteFile(instanceIDPath, []byte(instanceID), 0600)
	s.Require().NoError(err)

	instanceID, err = LoadInstanceID()
	s.Require().NoError(err)
	s.Equal(instanceID, instanceID)
}

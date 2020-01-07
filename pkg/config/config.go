package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gitlab.com/postgres-ai/database-lab/pkg/services/cloning"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/pkg/srv"

	"gopkg.in/yaml.v2"
)

// Config contains a common database-lab configuration.
type Config struct {
	Server    srv.Config       `yaml:"server"`
	Provision provision.Config `yaml:"provision"`
	Cloning   cloning.Config   `yaml:"cloning"`
	Debug     bool             `yaml:"debug"`
}

// LoadConfig instances a new Config by configuration filename.
func LoadConfig(name string) (*Config, error) {
	cfg := &Config{}

	path, err := getConfigPath(name)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error loading %s config file", name)
	}

	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s config", name)
	}

	return cfg, nil
}

func getConfigPath(name string) (string, error) {
	bindir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}

	dir, err := filepath.Abs(filepath.Dir(bindir))
	if err != nil {
		return "", err
	}

	path := dir + string(os.PathSeparator) + "config" +
		string(os.PathSeparator) + name

	return path, nil
}

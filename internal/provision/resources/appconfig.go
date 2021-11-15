/*
2020 © Postgres.ai
*/

package resources

import (
	"path"
)

// AppConfig currently stores Postgres configuration (other application in the future too).
type AppConfig struct {
	CloneName   string
	DockerImage string
	Pool        *Pool
	Host        string
	Port        uint
	DB          *DB
	NetworkID   string

	ContainerConf map[string]string
	pgExtraConf   map[string]string
}

// DB describes a default database configuration.
type DB struct {
	Username string
	DBName   string
}

// DataDir returns the path of clone data.
func (c *AppConfig) DataDir() string {
	// TODO(akartasov): Move to pool.
	return path.Join(c.Pool.ClonesDir(), c.CloneName, c.Pool.DataSubDir)
}

// ExtraConf returns a map with an extra configuration.
func (c *AppConfig) ExtraConf() map[string]string {
	return c.pgExtraConf
}

// SetExtraConf sets a map with an extra configuration.
func (c *AppConfig) SetExtraConf(extraConf map[string]string) {
	c.pgExtraConf = extraConf
}

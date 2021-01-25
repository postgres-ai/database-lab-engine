/*
2020 © Postgres.ai
*/

package resources

import (
	"path"
)

// AppConfig currently stores Postgres configuration (other application in the future too).
type AppConfig struct {
	CloneName string
	ClonePool string

	DockerImage string

	// PGDATA.
	MountDir           string
	DataSubDir         string
	ClonesMountDir     string
	UnixSocketCloneDir string

	Host string
	Port uint

	dbName string

	// The specified user must exist. The user will not be created automatically.
	username string
	password string

	OSUsername string

	ContainerConf map[string]string
	pgExtraConf   map[string]string
}

// Username return username defined in AppConfig or default value.
func (c *AppConfig) Username() string {
	if c.username != "" {
		return c.username
	}

	return "postgres"
}

// SetUsername sets username in AppConfig.
func (c *AppConfig) SetUsername(username string) {
	c.username = username
}

// Password return Password defined in AppConfig or default value.
func (c *AppConfig) Password() string {
	if c.password != "" {
		return c.password
	}

	return "postgres"
}

// SetPassword sets password in AppConfig
func (c *AppConfig) SetPassword(password string) {
	c.password = password
}

// DBName return a database name defined in AppConfig or default value.
func (c *AppConfig) DBName() string {
	if c.dbName != "" {
		return c.dbName
	}

	return "postgres"
}

// SetDBName sets dbName in AppConfig.
func (c *AppConfig) SetDBName(dbName string) {
	c.dbName = dbName
}

// DataDir returns the path of clone data.
func (c *AppConfig) DataDir() string {
	return path.Join(c.ClonesMountDir, c.CloneName, c.DataSubDir)
}

// ExtraConf returns a map with an extra configuration.
func (c *AppConfig) ExtraConf() map[string]string {
	return c.pgExtraConf
}

// SetExtraConf sets a map with an extra configuration.
func (c *AppConfig) SetExtraConf(extraConf map[string]string) {
	c.pgExtraConf = extraConf
}

/*
2019 © Postgres.ai
*/

// Package config provides access to the Database Lab configuration.
package config

import (
	"gitlab.com/postgres-ai/database-lab/v3/internal/cloning"
	"gitlab.com/postgres-ai/database-lab/v3/internal/diagnostic"
	"gitlab.com/postgres-ai/database-lab/v3/internal/embeddedui"
	"gitlab.com/postgres-ai/database-lab/v3/internal/observer"
	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	retConfig "gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	srvCfg "gitlab.com/postgres-ai/database-lab/v3/internal/srv/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/webhooks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
)

const (
	configName     = "server.yml"
	instanceIDFile = "instance_id"
)

// Config contains a common database-lab configuration.
type Config struct {
	Server      srvCfg.Config     `yaml:"server"`
	Provision   provision.Config  `yaml:"provision"`
	Cloning     cloning.Config    `yaml:"cloning"`
	Platform    platform.Config   `yaml:"platform"`
	Global      global.Config     `yaml:"global"`
	Retrieval   retConfig.Config  `yaml:"retrieval"`
	Observer    observer.Config   `yaml:"observer"`
	PoolManager pool.Config       `yaml:"poolManager"`
	EmbeddedUI  embeddedui.Config `yaml:"embeddedUI"`
	Diagnostic  diagnostic.Config `yaml:"diagnostic"`
	Webhooks    webhooks.Config   `yaml:"webhooks"`
}

/*
2021 © Postgres.ai
*/

// Package global provides access to the global Database Lab Engine configuration.
package global

import (
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/defaults"
)

// Config contains global Database Lab configurations.
type Config struct {
	Database Database `yaml:"database"`
	Engine   string   `yaml:"engine"`
	Debug    bool     `yaml:"debug"`
}

// Database contains default configurations of the managed database.
type Database struct {
	Username string `yaml:"username"`
	DBName   string `yaml:"dbname"`
}

// User returns default Database username.
func (d *Database) User() string {
	if d.Username != "" {
		return d.Username
	}

	return defaults.Username
}

// Name returns default Database name.
func (d *Database) Name() string {
	if d.DBName != "" {
		return d.DBName
	}

	return defaults.DBName
}

// EngineProps contains internal Database Lab Engine properties.
type EngineProps struct {
	InstanceID     string
	ContainerName  string
	Infrastructure string
	BillingActive  bool
	EnginePort     uint
}

const (
	// LocalInfra defines a local infra.
	LocalInfra = "local"

	// CommunityEdition defines the community edition.
	CommunityEdition = "community"

	// StandardEdition defines the standard edition.
	StandardEdition = "standard"

	// EnterpriseEdition defines the enterprise edition.
	EnterpriseEdition = "enterprise"

	// AWSInfrastructure marks instances running from AWS Marketplace.
	AWSInfrastructure = "AWS"
)

// GetEdition provides the DLE edition.
func (p *EngineProps) GetEdition() string {
	if p.Infrastructure != LocalInfra {
		return StandardEdition
	}

	return CommunityEdition
}

// UpdateBilling sets actual state of the billing activity.
func (p *EngineProps) UpdateBilling(activity bool) {
	p.BillingActive = activity
}

// CheckBilling checks the billing of the DLE instance is active.
func (p *EngineProps) CheckBilling() error {
	if p.Infrastructure == AWSInfrastructure {
		return nil
	}

	if !p.BillingActive {
		return errors.Errorf("billing is not active")
	}

	return nil
}

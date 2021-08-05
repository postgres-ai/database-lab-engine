/*
2019 © Postgres.ai
*/

// Package cloning provides a cloning service.
package cloning

import (
	"context"
	"time"

	"github.com/jackc/pgtype/pgxtype"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
)

// Config contains a cloning configuration.
type Config struct {
	MaxIdleMinutes uint   `yaml:"maxIdleMinutes"`
	AccessHost     string `yaml:"accessHost"`
}

type cloning struct {
	Config *Config
}

// Cloning defines a Cloning service interface.
type Cloning interface {
	Run(ctx context.Context) error
	Reload(config Config)

	CreateClone(*types.CloneCreateRequest) (*models.Clone, error)
	CloneConnection(ctx context.Context, cloneID string) (pgxtype.Querier, error)
	DestroyClone(string) error
	GetClone(string) (*models.Clone, error)
	UpdateClone(string, *types.CloneUpdateRequest) (*models.Clone, error)
	UpdateCloneStatus(string, models.Status) error
	ResetClone(string) error

	GetInstanceState() (*models.InstanceStatus, error)
	GetSnapshots() ([]models.Snapshot, error)
	GetClones() []*models.Clone
}

// CloneWrapper represents a cloning service wrapper.
type CloneWrapper struct {
	clone   *models.Clone
	session *resources.Session

	timeCreatedAt time.Time
	timeStartedAt time.Time

	username string
	password string

	snapshot models.Snapshot
}

// New returns a cloning interface depends on configuration mode.
// TODO: get rid of the interface, return *baseCloning instead.
func New(cfg *Config, provision *provision.Provisioner, observingCh chan string) Cloning {
	return NewBaseCloning(cfg, provision, observingCh)
}

// NewCloneWrapper constructs a new CloneWrapper.
func NewCloneWrapper(clone *models.Clone) *CloneWrapper {
	w := &CloneWrapper{
		clone: clone,
	}

	return w
}

// IsProtected checks if clone is protected.
func (cw CloneWrapper) IsProtected() bool {
	return cw.clone != nil && cw.clone.Protected
}

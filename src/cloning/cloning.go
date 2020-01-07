/*
2019 © Postgres.ai
*/

package cloning

import (
	"fmt"
	"time"

	"gitlab.com/postgres-ai/database-lab/src/log"
	"gitlab.com/postgres-ai/database-lab/src/models"
	"gitlab.com/postgres-ai/database-lab/src/provision"
)

const MODE_BASE = "base"
const MODE_MOCK = "mock"

type Config struct {
	Mode       string `yaml:"mode"`
	AutoDelete bool   `yaml:"autoDelete"`
	IdleTime   uint   `yaml:"idleTime"`
	AccessHost string `yaml:"accessHost"`
}

type cloning struct {
	Config *Config
}

type Cloning interface {
	Run() error

	CreateClone(*models.Clone) error
	DestroyClone(string) error
	GetClone(string) (*models.Clone, bool)
	UpdateClone(string, *models.Clone) error
	ResetClone(string) error

	GetInstanceState() (*models.InstanceStatus, error)
	GetSnapshots() ([]*models.Snapshot, error)
	GetClones() []*models.Clone
}

type CloneWrapper struct {
	clone   *models.Clone
	session *provision.Session

	timeCreatedAt time.Time
	timeStartedAt time.Time

	username string
	password string

	snapshot *models.Snapshot
}

func NewCloning(config *Config, provision provision.Provision) (Cloning, error) {
	switch config.Mode {
	case "", MODE_BASE:
		log.Dbg("Using base cloning mode.")
		return NewBaseCloning(config, provision), nil
	case MODE_MOCK:
		log.Dbg("Using mock cloning mode.")
		return nil, nil
	}

	return nil, fmt.Errorf("Unsupported mode specified.")
}

func NewCloneWrapper(clone *models.Clone) *CloneWrapper {
	w := &CloneWrapper{
		clone: clone,
	}

	if clone.Db == nil {
		clone.Db = &models.Database{}
	}

	return w
}

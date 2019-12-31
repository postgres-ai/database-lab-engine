/*
2019 © Postgres.ai
*/

package cloning

import (
	"fmt"
	"time"

	"../log"
	m "../models"
	p "../provision"
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

	CreateClone(*m.Clone) error
	DestroyClone(string) error
	GetClone(string) (*m.Clone, bool)
	UpdateClone(string, *m.Clone) error
	ResetClone(string) error

	GetInstanceState() (*m.InstanceStatus, error)
	GetSnapshots() ([]*m.Snapshot, error)
	GetClones() []*m.Clone
}

type CloneWrapper struct {
	clone   *m.Clone
	session *p.Session

	timeCreatedAt time.Time
	timeStartedAt time.Time

	username string
	password string

	snapshot *m.Snapshot
}

func NewCloning(config *Config, provision p.Provision) (Cloning, error) {
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

func NewCloneWrapper(clone *m.Clone) *CloneWrapper {
	w := &CloneWrapper{
		clone: clone,
	}

	if clone.Db == nil {
		clone.Db = &m.Database{}
	}

	return w
}

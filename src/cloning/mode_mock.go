/*
2019 © Postgres.ai
*/

package cloning

import (
	"fmt"

	"../log"
	m "../models"
)

type mockCloning struct {
	cloning

	clones         map[string]*m.Clone
	instanceStatus *m.InstanceStatus
	snapshots      []*m.Snapshot
}

func NewMockCloning(cfg *Config) Cloning {
	var instanceStatusActualStatus = &m.Status{
		Code:    "OK",
		Message: "Instance is ready",
	}

	var fs = &m.FileSystem{}

	var instanceStatus = m.InstanceStatus{
		Status:     instanceStatusActualStatus,
		FileSystem: fs,
		Clones:     make([]*m.Clone, 0),
	}

	cloning := &mockCloning{}
	cloning.Config = cfg
	cloning.clones = make(map[string]*m.Clone)
	cloning.instanceStatus = &instanceStatus

	return cloning
}

func NewMockClone() *m.Clone {
	db := &m.Database{}
	return &m.Clone{
		Id:          "id",
		Name:        "name",
		Snapshot:    "snapshot",
		CloneSize:   1000,
		CloningTime: 10.0,
		Protected:   false,
		DeleteAt:    "10000",
		CreatedAt:   "10000",
		Status:      statusOk,
		Db:          db,
	}
}

func (c *mockCloning) Run() error {
	return nil
}

func (c *mockCloning) CreateClone(clone *m.Clone) error {
	if len(clone.Name) == 0 {
		return fmt.Errorf("Missing required fields.")
	}
	return nil
}

func (c *mockCloning) DestroyClone(id string) error {
	_, ok := c.clones[id]
	if !ok {
		err := fmt.Errorf("Clone not found.")
		log.Err(err)
		return err
	}

	return nil
}

func (c *mockCloning) GetClone(id string) (*m.Clone, bool) {
	clone, ok := c.clones[id]
	return clone, ok
}

func (c *mockCloning) UpdateClone(id string, patch *m.Clone) error {
	_, ok := c.clones[id]
	if !ok {
		err := fmt.Errorf("Clone not found.")
		log.Err(err)
		return err
	}

	return nil
}

func (c *mockCloning) ResetClone(id string) error {
	_, ok := c.clones[id]
	if !ok {
		err := fmt.Errorf("Clone not found.")
		log.Err(err)
		return err
	}

	return nil
}

func (c *mockCloning) GetInstanceState() (*m.InstanceStatus, error) {
	return c.instanceStatus, nil
}

func (c *mockCloning) GetSnapshots() []*m.Snapshot {
	return c.snapshots
}

func (c *mockCloning) GetClones() []*m.Clone {
	clones := make([]*m.Clone, 0)
	for _, clone := range c.clones {
		clones = append(clones, clone)
	}
	return clones
}

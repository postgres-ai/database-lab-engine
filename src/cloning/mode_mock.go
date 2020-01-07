/*
2019 © Postgres.ai
*/

package cloning

import (
	"fmt"

	"gitlab.com/postgres-ai/database-lab/src/log"
	"gitlab.com/postgres-ai/database-lab/src/models"
)

type mockCloning struct {
	cloning

	clones         map[string]*models.Clone
	instanceStatus *models.InstanceStatus
	snapshots      []*models.Snapshot
}

func NewMockCloning(cfg *Config) Cloning {
	var instanceStatusActualStatus = &models.Status{
		Code:    "OK",
		Message: "Instance is ready",
	}

	var fs = &models.FileSystem{}

	var instanceStatus = models.InstanceStatus{
		Status:     instanceStatusActualStatus,
		FileSystem: fs,
		Clones:     make([]*models.Clone, 0),
	}

	cloning := &mockCloning{}
	cloning.Config = cfg
	cloning.clones = make(map[string]*models.Clone)
	cloning.instanceStatus = &instanceStatus

	return cloning
}

func NewMockClone() *models.Clone {
	db := &models.Database{}
	snapshot := &models.Snapshot{}

	return &models.Clone{
		Id:          "id",
		Name:        "name",
		Snapshot:    snapshot,
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

func (c *mockCloning) CreateClone(clone *models.Clone) error {
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

func (c *mockCloning) GetClone(id string) (*models.Clone, bool) {
	clone, ok := c.clones[id]
	return clone, ok
}

func (c *mockCloning) UpdateClone(id string, patch *models.Clone) error {
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

func (c *mockCloning) GetInstanceState() (*models.InstanceStatus, error) {
	return c.instanceStatus, nil
}

func (c *mockCloning) GetSnapshots() ([]*models.Snapshot, error) {
	return c.snapshots, nil
}

func (c *mockCloning) GetClones() []*models.Clone {
	clones := make([]*models.Clone, 0)
	for _, clone := range c.clones {
		clones = append(clones, clone)
	}
	return clones
}

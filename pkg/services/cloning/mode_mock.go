/*
2019 © Postgres.ai
*/

package cloning

import (
	"context"

	"github.com/jackc/pgtype/pgxtype"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
)

type mockCloning struct {
	cloning

	clones         map[string]*models.Clone
	instanceStatus *models.InstanceStatus
	snapshots      []models.Snapshot
}

// NewMockCloning instances a new mock Cloning.
func NewMockCloning(cfg *Config) Cloning {
	var instanceStatusActualStatus = &models.Status{
		Code:    models.StatusOK,
		Message: models.InstanceMessageOK,
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

const (
	mockCloneSize      = 1000
	mockCloningTime    = 10.0
	mockMaxIdleMinutes = 120
)

// NewMockClone instances a new Clone model.
func NewMockClone() *models.Clone {
	db := models.Database{}
	snapshot := &models.Snapshot{}

	return &models.Clone{
		ID:       "id",
		Snapshot: snapshot,
		Metadata: models.CloneMetadata{
			CloneDiffSize:  mockCloneSize,
			CloningTime:    mockCloningTime,
			MaxIdleMinutes: mockMaxIdleMinutes,
		},
		Protected: false,
		DeleteAt:  "10000",
		CreatedAt: "10000",
		Status: models.Status{
			Code:    models.StatusOK,
			Message: models.CloneMessageOK,
		},
		DB: db,
	}
}

func (c *mockCloning) Reload(_ Config) {}

func (c *mockCloning) Run(ctx context.Context) error {
	return nil
}

func (c *mockCloning) CreateClone(clone *types.CloneCreateRequest) (*models.Clone, error) {
	return &models.Clone{}, nil
}

func (c *mockCloning) CloneConnection(ctx context.Context, cloneID string) (pgxtype.Querier, error) {
	return nil, nil
}

func (c *mockCloning) DestroyClone(id string) error {
	if _, ok := c.clones[id]; !ok {
		return errors.New("clone not found")
	}

	return nil
}

func (c *mockCloning) GetClone(id string) (*models.Clone, error) {
	clone, ok := c.clones[id]
	if !ok {
		return nil, errors.New("clone not found")
	}

	return clone, nil
}

func (c *mockCloning) UpdateClone(id string, _ types.CloneUpdateRequest) (*models.Clone, error) {
	if _, ok := c.clones[id]; !ok {
		return nil, errors.New("clone not found")
	}

	return &models.Clone{}, nil
}

func (c *mockCloning) UpdateCloneStatus(id string, _ models.Status) error {
	if _, ok := c.clones[id]; !ok {
		return errors.New("clone not found")
	}

	return nil
}

func (c *mockCloning) ResetClone(id string, _ types.ResetCloneRequest) error {
	if _, ok := c.clones[id]; !ok {
		return errors.New("clone not found")
	}

	return nil
}

func (c *mockCloning) GetInstanceState() (*models.InstanceStatus, error) {
	return c.instanceStatus, nil
}

func (c *mockCloning) GetSnapshots() ([]models.Snapshot, error) {
	return c.snapshots, nil
}

func (c *mockCloning) GetClones() []*models.Clone {
	clones := make([]*models.Clone, 0, len(c.clones))
	for _, clone := range c.clones {
		clones = append(clones, clone)
	}

	return clones
}

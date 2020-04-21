package cloning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/postgres-ai/database-lab/pkg/models"
)

func TestBaseCloningSuite(t *testing.T) {
	suite.Run(t, new(BaseCloningSuite))
}

type BaseCloningSuite struct {
	cloning *baseCloning

	suite.Suite
}

func (s *BaseCloningSuite) SetupSuite() {
	cloning := &baseCloning{
		clones:    make(map[string]*CloneWrapper),
		snapshots: make([]models.Snapshot, 0),
	}

	s.cloning = cloning
}

func (s *BaseCloningSuite) TearDownTest() {
	s.cloning.clones = make(map[string]*CloneWrapper)
	s.cloning.snapshots = make([]models.Snapshot, 0)
}

func (s *BaseCloningSuite) TestFindWrapper() {
	wrapper, ok := s.cloning.findWrapper("testCloneID")
	assert.False(s.T(), ok)
	assert.Nil(s.T(), wrapper)

	s.cloning.setWrapper("testCloneID", &CloneWrapper{clone: &models.Clone{ID: "testCloneID"}})

	wrapper, ok = s.cloning.findWrapper("testCloneID")
	assert.True(s.T(), ok)
	assert.NotNil(s.T(), wrapper)
	assert.Equal(s.T(), CloneWrapper{clone: &models.Clone{ID: "testCloneID"}}, *wrapper)
}

func (s *BaseCloningSuite) TestUpdateStatus() {
	s.cloning.setWrapper("testCloneID", &CloneWrapper{clone: &models.Clone{Status: models.Status{
		Code:    models.StatusCreating,
		Message: models.CloneMessageCreating,
	}}})

	wrapper, ok := s.cloning.findWrapper("testCloneID")
	require.True(s.T(), ok)
	require.NotNil(s.T(), wrapper)

	err := s.cloning.updateCloneStatus("testCloneID", models.Status{
		Code:    models.StatusOK,
		Message: models.CloneMessageOK,
	})
	require.NoError(s.T(), err)

	wrapper, ok = s.cloning.findWrapper("testCloneID")
	require.True(s.T(), ok)
	require.NotNil(s.T(), wrapper)

	assert.Equal(s.T(), models.Status{
		Code:    models.StatusOK,
		Message: models.CloneMessageOK,
	}, wrapper.clone.Status)
}

func (s *BaseCloningSuite) TestDeleteClone() {
	wrapper, ok := s.cloning.findWrapper("testCloneID")
	assert.False(s.T(), ok)
	assert.Nil(s.T(), wrapper)

	s.cloning.setWrapper("testCloneID", &CloneWrapper{})

	wrapper, ok = s.cloning.findWrapper("testCloneID")
	require.True(s.T(), ok)
	require.NotNil(s.T(), wrapper)
	assert.Equal(s.T(), CloneWrapper{}, *wrapper)

	s.cloning.deleteClone("testCloneID")

	wrapper, ok = s.cloning.findWrapper("testCloneID")
	assert.False(s.T(), ok)
	assert.Nil(s.T(), wrapper)
}

func (s *BaseCloningSuite) TestLenClones() {
	lenClones := s.cloning.lenClones()
	assert.Equal(s.T(), 0, lenClones)

	s.cloning.setWrapper("testCloneID1", &CloneWrapper{})
	s.cloning.setWrapper("testCloneID2", &CloneWrapper{})

	lenClones = s.cloning.lenClones()
	assert.Equal(s.T(), 2, lenClones)

	s.cloning.deleteClone("testCloneID1")

	lenClones = s.cloning.lenClones()
	assert.Equal(s.T(), 1, lenClones)
}

func (s *BaseCloningSuite) TestLatestSnapshot() {
	snapshot1 := models.Snapshot{
		ID:          "TestSnapshotID1",
		CreatedAt:   "2020-02-20 01:23:45",
		DataStateAt: "2020-02-19 00:00:00",
	}

	snapshot2 := models.Snapshot{
		ID:          "TestSnapshotID2",
		CreatedAt:   "2020-02-20 05:43:21",
		DataStateAt: "2020-02-20 00:00:00",
	}

	assert.Equal(s.T(), 0, len(s.cloning.snapshots))
	latestSnapshot, err := s.cloning.getLatestSnapshot()
	require.Equal(s.T(), latestSnapshot, models.Snapshot{})
	assert.EqualError(s.T(), err, "no snapshot found")

	s.cloning.snapshots = append(s.cloning.snapshots, snapshot1, snapshot2)

	latestSnapshot, err = s.cloning.getLatestSnapshot()
	require.NoError(s.T(), err)
	assert.Equal(s.T(), latestSnapshot, snapshot1)
}

func (s *BaseCloningSuite) TestSnapshotByID() {
	snapshot1 := models.Snapshot{
		ID:          "TestSnapshotID1",
		CreatedAt:   "2020-02-20 01:23:45",
		DataStateAt: "2020-02-19 00:00:00",
	}

	snapshot2 := models.Snapshot{
		ID:          "TestSnapshotID2",
		CreatedAt:   "2020-02-20 05:43:21",
		DataStateAt: "2020-02-20 00:00:00",
	}

	assert.Equal(s.T(), 0, len(s.cloning.snapshots))
	latestSnapshot, err := s.cloning.getLatestSnapshot()
	require.Equal(s.T(), latestSnapshot, models.Snapshot{})
	assert.EqualError(s.T(), err, "no snapshot found")

	s.cloning.snapshots = append(s.cloning.snapshots, snapshot1, snapshot2)

	latestSnapshot, err = s.cloning.getSnapshotByID("TestSnapshotID2")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), latestSnapshot, snapshot2)
}

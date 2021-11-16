/*
2021 © Postgres.ai
*/

package cloning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
)

func TestBaseCloningSuite(t *testing.T) {
	suite.Run(t, new(BaseCloningSuite))
}

type BaseCloningSuite struct {
	cloning *Base

	suite.Suite
}

func (s *BaseCloningSuite) SetupSuite() {
	cloning := &Base{
		clones:      make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	s.cloning = cloning
}

func (s *BaseCloningSuite) TearDownTest() {
	s.cloning.clones = make(map[string]*CloneWrapper)
	s.cloning.snapshotBox = SnapshotBox{items: make(map[string]*models.Snapshot)}
}

func (s *BaseCloningSuite) TestFindWrapper() {
	wrapper, ok := s.cloning.findWrapper("testCloneID")
	assert.False(s.T(), ok)
	assert.Nil(s.T(), wrapper)

	s.cloning.setWrapper("testCloneID", &CloneWrapper{Clone: &models.Clone{ID: "testCloneID"}})

	wrapper, ok = s.cloning.findWrapper("testCloneID")
	assert.True(s.T(), ok)
	assert.NotNil(s.T(), wrapper)
	assert.Equal(s.T(), CloneWrapper{Clone: &models.Clone{ID: "testCloneID"}}, *wrapper)
}

func (s *BaseCloningSuite) TestCloneList() {
	clone1 := &models.Clone{CreatedAt: "2020-02-20 01:23:45 UTC"}
	clone2 := &models.Clone{CreatedAt: "2020-06-23 10:31:27 UTC"}
	clone3 := &models.Clone{CreatedAt: "2020-05-20 00:43:21 UTC"}

	s.cloning.setWrapper("testCloneID1", &CloneWrapper{Clone: clone1})
	s.cloning.setWrapper("testCloneID2", &CloneWrapper{Clone: clone2})
	s.cloning.setWrapper("testCloneID3", &CloneWrapper{Clone: clone3})

	list := s.cloning.GetClones()

	assert.Equal(s.T(), 3, len(list))

	// Check clone order.
	assert.Equal(s.T(), []*models.Clone{
		{CreatedAt: "2020-06-23 10:31:27 UTC"},
		{CreatedAt: "2020-05-20 00:43:21 UTC"},
		{CreatedAt: "2020-02-20 01:23:45 UTC"},
	}, list)
}

func (s *BaseCloningSuite) TestUpdateStatus() {
	s.cloning.setWrapper("testCloneID", &CloneWrapper{Clone: &models.Clone{Status: models.Status{
		Code:    models.StatusCreating,
		Message: models.CloneMessageCreating,
	}}})

	wrapper, ok := s.cloning.findWrapper("testCloneID")
	require.True(s.T(), ok)
	require.NotNil(s.T(), wrapper)

	err := s.cloning.UpdateCloneStatus("testCloneID", models.Status{
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
	}, wrapper.Clone.Status)
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

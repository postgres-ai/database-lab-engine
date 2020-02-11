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
		clones: make(map[string]*CloneWrapper),
	}

	s.cloning = cloning
}

func (s *BaseCloningSuite) TearDownTest() {
	s.cloning.clones = make(map[string]*CloneWrapper)
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
	s.cloning.setWrapper("testCloneID", &CloneWrapper{clone: &models.Clone{Status: &models.Status{
		Code:    models.StatusCreating,
		Message: models.CloneMessageCreating,
	}}})

	wrapper, ok := s.cloning.findWrapper("testCloneID")
	require.True(s.T(), ok)
	require.NotNil(s.T(), wrapper)

	s.cloning.updateStatus(wrapper.clone, models.Status{
		Code:    models.StatusOK,
		Message: models.CloneMessageOK,
	})

	wrapper, ok = s.cloning.findWrapper("testCloneID")
	require.True(s.T(), ok)
	require.NotNil(s.T(), wrapper)

	assert.Equal(s.T(), models.Status{
		Code:    models.StatusOK,
		Message: models.CloneMessageOK,
	}, *wrapper.clone.Status)
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

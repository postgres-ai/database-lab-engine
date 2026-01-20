/*
2021 © Postgres.ai
*/

package cloning

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
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
	clone1 := &models.Clone{CreatedAt: &models.LocalTime{Time: time.Date(2020, 02, 20, 01, 23, 45, 0, time.UTC)}}
	clone2 := &models.Clone{CreatedAt: &models.LocalTime{Time: time.Date(2020, 06, 23, 10, 31, 27, 0, time.UTC)}}
	clone3 := &models.Clone{CreatedAt: &models.LocalTime{Time: time.Date(2020, 05, 20, 00, 43, 21, 0, time.UTC)}}

	s.cloning.setWrapper("testCloneID1", &CloneWrapper{Clone: clone1})
	s.cloning.setWrapper("testCloneID2", &CloneWrapper{Clone: clone2})
	s.cloning.setWrapper("testCloneID3", &CloneWrapper{Clone: clone3})

	list := s.cloning.GetClones()

	assert.Equal(s.T(), 3, len(list))

	// Check clone order.
	assert.Equal(s.T(), []*models.Clone{
		{CreatedAt: &models.LocalTime{Time: time.Date(2020, 06, 23, 10, 31, 27, 0, time.UTC)}},
		{CreatedAt: &models.LocalTime{Time: time.Date(2020, 05, 20, 00, 43, 21, 0, time.UTC)}},
		{CreatedAt: &models.LocalTime{Time: time.Date(2020, 02, 20, 01, 23, 45, 0, time.UTC)}},
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

func TestCalculateProtectionTime(t *testing.T) {
	tests := []struct {
		name            string
		config          Config
		durationMinutes *uint
		expectNil       bool
		minExpiry       time.Duration
		maxExpiry       time.Duration
	}{
		{name: "nil duration uses default", config: Config{ProtectionLeaseDurationMinutes: 60}, durationMinutes: nil, expectNil: false, minExpiry: 59 * time.Minute, maxExpiry: 61 * time.Minute},
		{name: "explicit duration overrides default", config: Config{ProtectionLeaseDurationMinutes: 60}, durationMinutes: ptrUint(120), expectNil: false, minExpiry: 119 * time.Minute, maxExpiry: 121 * time.Minute},
		{name: "zero duration with no max returns nil (forever)", config: Config{ProtectionLeaseDurationMinutes: 60, ProtectionMaxDurationMinutes: 0}, durationMinutes: ptrUint(0), expectNil: true},
		{name: "zero duration with max uses max", config: Config{ProtectionLeaseDurationMinutes: 60, ProtectionMaxDurationMinutes: 120}, durationMinutes: ptrUint(0), expectNil: false, minExpiry: 119 * time.Minute, maxExpiry: 121 * time.Minute},
		{name: "duration exceeding max is capped", config: Config{ProtectionLeaseDurationMinutes: 60, ProtectionMaxDurationMinutes: 120}, durationMinutes: ptrUint(300), expectNil: false, minExpiry: 119 * time.Minute, maxExpiry: 121 * time.Minute},
		{name: "duration within max is used", config: Config{ProtectionLeaseDurationMinutes: 60, ProtectionMaxDurationMinutes: 300}, durationMinutes: ptrUint(120), expectNil: false, minExpiry: 119 * time.Minute, maxExpiry: 121 * time.Minute},
		{name: "default zero duration with no max returns nil", config: Config{ProtectionLeaseDurationMinutes: 0, ProtectionMaxDurationMinutes: 0}, durationMinutes: nil, expectNil: true},
		{name: "default zero duration with max uses max", config: Config{ProtectionLeaseDurationMinutes: 0, ProtectionMaxDurationMinutes: 60}, durationMinutes: nil, expectNil: false, minExpiry: 59 * time.Minute, maxExpiry: 61 * time.Minute},
		{name: "1 minute duration", config: Config{}, durationMinutes: ptrUint(1), expectNil: false, minExpiry: 50 * time.Second, maxExpiry: 70 * time.Second},
		{name: "7 days duration", config: Config{}, durationMinutes: ptrUint(10080), expectNil: false, minExpiry: 10079 * time.Minute, maxExpiry: 10081 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &Base{config: &tt.config}
			result := base.calculateProtectionTime(tt.durationMinutes)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				expiresIn := time.Until(result.Time)
				assert.GreaterOrEqual(t, expiresIn, tt.minExpiry)
				assert.LessOrEqual(t, expiresIn, tt.maxExpiry)
			}
		})
	}
}

func TestCalculateProtectionTime_EdgeCases(t *testing.T) {
	t.Run("max equals requested duration", func(t *testing.T) {
		base := &Base{config: &Config{ProtectionMaxDurationMinutes: 60}}
		result := base.calculateProtectionTime(ptrUint(60))

		require.NotNil(t, result)
		expiresIn := time.Until(result.Time)
		assert.GreaterOrEqual(t, expiresIn, 59*time.Minute)
		assert.LessOrEqual(t, expiresIn, 61*time.Minute)
	})

	t.Run("max is 1 minute and request is 1 minute", func(t *testing.T) {
		base := &Base{config: &Config{ProtectionMaxDurationMinutes: 1}}
		result := base.calculateProtectionTime(ptrUint(1))

		require.NotNil(t, result)
		expiresIn := time.Until(result.Time)
		assert.Greater(t, expiresIn, time.Duration(0))
		assert.LessOrEqual(t, expiresIn, 2*time.Minute)
	})

	t.Run("very large duration is capped by max", func(t *testing.T) {
		base := &Base{config: &Config{ProtectionMaxDurationMinutes: 60}}
		result := base.calculateProtectionTime(ptrUint(999999))

		require.NotNil(t, result)
		expiresIn := time.Until(result.Time)
		assert.GreaterOrEqual(t, expiresIn, 59*time.Minute)
		assert.LessOrEqual(t, expiresIn, 61*time.Minute)
	})

	t.Run("nil config protection lease with explicit duration", func(t *testing.T) {
		base := &Base{config: &Config{}}
		result := base.calculateProtectionTime(ptrUint(30))

		require.NotNil(t, result)
		expiresIn := time.Until(result.Time)
		assert.GreaterOrEqual(t, expiresIn, 29*time.Minute)
		assert.LessOrEqual(t, expiresIn, 31*time.Minute)
	})
}

func ptrUint(v uint) *uint {
	return &v
}

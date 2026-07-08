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
	s.cloning.resetClones()
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)
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

func (s *BaseCloningSuite) TestGetExpectedCloningTime() {
	t := s.T()

	result := s.cloning.getExpectedCloningTime()
	assert.Equal(t, 0.0, result, "expected zero when no clones exist")

	s.cloning.setWrapper("clone1", &CloneWrapper{Clone: &models.Clone{Metadata: models.CloneMetadata{CloningTime: 2.0}}})
	s.cloning.setWrapper("clone2", &CloneWrapper{Clone: &models.Clone{Metadata: models.CloneMetadata{CloningTime: 4.0}}})
	s.cloning.setWrapper("clone3", &CloneWrapper{Clone: &models.Clone{Metadata: models.CloneMetadata{CloningTime: 6.0}}})

	result = s.cloning.getExpectedCloningTime()
	assert.Equal(t, 4.0, result, "expected average of 2, 4, 6")
}

func (s *BaseCloningSuite) TestGetExpectedCloningTimeSingleClone() {
	t := s.T()

	s.cloning.setWrapper("clone1", &CloneWrapper{Clone: &models.Clone{Metadata: models.CloneMetadata{CloningTime: 3.5}}})

	result := s.cloning.getExpectedCloningTime()
	assert.Equal(t, 3.5, result)
}

func (s *BaseCloningSuite) TestGetExpectedCloningTimeZeroValues() {
	t := s.T()

	s.cloning.setWrapper("clone1", &CloneWrapper{Clone: &models.Clone{Metadata: models.CloneMetadata{CloningTime: 0.0}}})
	s.cloning.setWrapper("clone2", &CloneWrapper{Clone: &models.Clone{Metadata: models.CloneMetadata{CloningTime: 0.0}}})

	result := s.cloning.getExpectedCloningTime()
	assert.Equal(t, 0.0, result)
}

func (s *BaseCloningSuite) TestUpdateCloneStatusNotFound() {
	t := s.T()

	err := s.cloning.UpdateCloneStatus("nonexistent", models.Status{Code: models.StatusOK, Message: models.CloneMessageOK})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func (s *BaseCloningSuite) TestUpdateCloneSnapshot() {
	t := s.T()

	originalSnapshot := &models.Snapshot{ID: "snap1"}
	newSnapshot := &models.Snapshot{ID: "snap2"}

	s.cloning.setWrapper("clone1", &CloneWrapper{Clone: &models.Clone{ID: "clone1", Snapshot: originalSnapshot}})

	err := s.cloning.UpdateCloneSnapshot("clone1", newSnapshot)
	require.NoError(t, err)

	w, ok := s.cloning.findWrapper("clone1")
	require.True(t, ok)
	assert.Equal(t, "snap2", w.Clone.Snapshot.ID)
}

func (s *BaseCloningSuite) TestUpdateCloneSnapshotNotFound() {
	t := s.T()

	err := s.cloning.UpdateCloneSnapshot("nonexistent", &models.Snapshot{ID: "snap1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     string
		username string
		dbname   string
		expected string
	}{
		{name: "standard values", host: "localhost", port: "5432", username: "postgres", dbname: "testdb", expected: "host='localhost' port=5432 user='postgres' database='testdb'"},
		{name: "custom host and port", host: "10.0.0.1", port: "6000", username: "admin", dbname: "mydb", expected: "host='10.0.0.1' port=6000 user='admin' database='mydb'"},
		{name: "socket path host", host: "/var/run/postgres", port: "5433", username: "user1", dbname: "db1", expected: "host='/var/run/postgres' port=5433 user='user1' database='db1'"},
		{name: "empty dbname", host: "localhost", port: "5432", username: "postgres", dbname: "", expected: "host='localhost' port=5432 user='postgres' database=''"},
		{name: "dbname with single quote", host: "localhost", port: "5432", username: "postgres", dbname: "test'db", expected: "host='localhost' port=5432 user='postgres' database='test''db'"},
		{name: "dbname with spaces", host: "localhost", port: "5432", username: "postgres", dbname: "my database", expected: "host='localhost' port=5432 user='postgres' database='my database'"},
		{name: "dbname with backslash", host: "localhost", port: "5432", username: "postgres", dbname: `test\db`, expected: `host='localhost' port=5432 user='postgres' database='test\\db'`},
		{name: "username with single quote", host: "localhost", port: "5432", username: "user'name", dbname: "testdb", expected: `host='localhost' port=5432 user='user''name' database='testdb'`},
		{name: "username with backslash", host: "localhost", port: "5432", username: `user\name`, dbname: "testdb", expected: `host='localhost' port=5432 user='user\\name' database='testdb'`},
		{name: "combined quote and backslash in dbname", host: "localhost", port: "5432", username: "postgres", dbname: `test\'db`, expected: `host='localhost' port=5432 user='postgres' database='test\\''db'`},
		{name: "host with single quote", host: "host'name", port: "5432", username: "postgres", dbname: "testdb", expected: `host='host''name' port=5432 user='postgres' database='testdb'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connectionString(tt.host, tt.port, tt.username, tt.dbname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCloningState(t *testing.T) {
	cfg := &Config{ProtectionLeaseDurationMinutes: 60, ProtectionMaxDurationMinutes: 120}
	base := &Base{
		config:      cfg,
		clones:      make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	state := base.GetCloningState()
	assert.Equal(t, uint64(0), state.NumClones)
	assert.Equal(t, 0.0, state.ExpectedCloningTime)
	assert.Equal(t, uint(60), state.ProtectionLeaseDurationMinutes)
	assert.Equal(t, uint(120), state.ProtectionMaxDurationMinutes)
	assert.Empty(t, state.Clones)

	base.setWrapper("c1", &CloneWrapper{Clone: &models.Clone{
		CreatedAt: &models.LocalTime{Time: time.Now()},
		Metadata:  models.CloneMetadata{CloningTime: 1.5},
	}})

	state = base.GetCloningState()
	assert.Equal(t, uint64(1), state.NumClones)
	assert.Equal(t, 1.5, state.ExpectedCloningTime)
}

func TestWithBranchDeletionLock(t *testing.T) {
	c := &Base{
		clones: map[string]*CloneWrapper{
			"clone1": {Clone: &models.Clone{ID: "clone1", Snapshot: &models.Snapshot{ID: "pool@snap2"}}},
		},
	}

	t.Run("runs destroy when no listed snapshot has a registered clone", func(t *testing.T) {
		called := false

		err := c.WithBranchDeletionLock([]string{"pool@snap1"}, func() error {
			called = true
			return nil
		})

		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("refuses and skips destroy when a listed snapshot has a registered clone", func(t *testing.T) {
		called := false

		err := c.WithBranchDeletionLock([]string{"pool@snap1", "pool@snap2"}, func() error {
			called = true
			return nil
		})

		require.Error(t, err)
		assert.False(t, called)
		assert.Contains(t, err.Error(), "dependent clone")
	})
}

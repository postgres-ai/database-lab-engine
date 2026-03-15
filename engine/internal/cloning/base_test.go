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

func TestConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     string
		username string
		dbname   string
		want     string
	}{
		{name: "standard connection", host: "localhost", port: "5432", username: "postgres", dbname: "testdb", want: "host=localhost port=5432 user=postgres database='testdb'"},
		{name: "socket connection", host: "/var/run/postgresql", port: "6000", username: "user", dbname: "mydb", want: "host=/var/run/postgresql port=6000 user=user database='mydb'"},
		{name: "database name with spaces", host: "localhost", port: "5432", username: "admin", dbname: "my database", want: "host=localhost port=5432 user=admin database='my database'"},
		{name: "empty database name", host: "localhost", port: "5432", username: "postgres", dbname: "", want: "host=localhost port=5432 user=postgres database=''"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := connectionString(tt.host, tt.port, tt.username, tt.dbname)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetClone(t *testing.T) {
	base := &Base{
		config: &Config{},
		clones: make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	t.Run("existing clone returns clone", func(t *testing.T) {
		base.setWrapper("clone-1", &CloneWrapper{Clone: &models.Clone{ID: "clone-1", Status: models.Status{Code: models.StatusOK}}})
		clone, err := base.GetClone("clone-1")
		require.NoError(t, err)
		require.NotNil(t, clone)
		assert.Equal(t, "clone-1", clone.ID)
	})

	t.Run("non-existing clone returns error", func(t *testing.T) {
		clone, err := base.GetClone("nonexistent")
		require.Error(t, err)
		assert.Nil(t, clone)
		assert.Contains(t, err.Error(), "clone not found")
	})
}

func TestUpdateCloneStatus_ErrorPaths(t *testing.T) {
	base := &Base{
		config: &Config{},
		clones: make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	t.Run("update nonexistent clone returns error", func(t *testing.T) {
		err := base.UpdateCloneStatus("nonexistent", models.Status{Code: models.StatusOK})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("update existing clone succeeds", func(t *testing.T) {
		base.setWrapper("clone-1", &CloneWrapper{Clone: &models.Clone{ID: "clone-1"}})
		err := base.UpdateCloneStatus("clone-1", models.Status{Code: models.StatusOK, Message: "running"})
		require.NoError(t, err)

		w, ok := base.findWrapper("clone-1")
		require.True(t, ok)
		assert.Equal(t, models.StatusOK, w.Clone.Status.Code)
	})
}

func TestUpdateCloneSnapshot_ErrorPaths(t *testing.T) {
	base := &Base{
		config: &Config{},
		clones: make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	t.Run("update nonexistent clone returns error", func(t *testing.T) {
		err := base.UpdateCloneSnapshot("nonexistent", &models.Snapshot{ID: "snap-1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("update existing clone sets snapshot", func(t *testing.T) {
		base.setWrapper("clone-1", &CloneWrapper{Clone: &models.Clone{ID: "clone-1"}})
		snap := &models.Snapshot{ID: "snap-1"}
		err := base.UpdateCloneSnapshot("clone-1", snap)
		require.NoError(t, err)

		w, ok := base.findWrapper("clone-1")
		require.True(t, ok)
		assert.Equal(t, "snap-1", w.Clone.Snapshot.ID)
	})
}

func TestGetCloningState(t *testing.T) {
	base := &Base{
		config: &Config{ProtectionLeaseDurationMinutes: 60, ProtectionMaxDurationMinutes: 120},
		clones: make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	t.Run("empty state returns zero clones", func(t *testing.T) {
		state := base.GetCloningState()
		assert.Equal(t, uint64(0), state.NumClones)
		assert.Empty(t, state.Clones)
		assert.Equal(t, uint(60), state.ProtectionLeaseDurationMinutes)
		assert.Equal(t, uint(120), state.ProtectionMaxDurationMinutes)
	})

	t.Run("state reflects clone count", func(t *testing.T) {
		base.setWrapper("c1", &CloneWrapper{Clone: &models.Clone{ID: "c1", CreatedAt: &models.LocalTime{Time: time.Now()}}})
		base.setWrapper("c2", &CloneWrapper{Clone: &models.Clone{ID: "c2", CreatedAt: &models.LocalTime{Time: time.Now()}}})
		state := base.GetCloningState()
		assert.Equal(t, uint64(2), state.NumClones)
		assert.Len(t, state.Clones, 2)
	})
}

func TestDestroyPreChecks(t *testing.T) {
	t.Run("protected clone returns error", func(t *testing.T) {
		base := &Base{
			config: &Config{},
			clones: make(map[string]*CloneWrapper),
			snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
		}

		w := &CloneWrapper{Clone: &models.Clone{ID: "protected", Protected: true, Status: models.Status{Code: models.StatusOK}}}
		base.setWrapper("protected", w)

		err := base.destroyPreChecks("protected", w)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "protected")
	})

	t.Run("fatal protected clone bypasses protection check", func(t *testing.T) {
		base := &Base{
			config: &Config{},
			clones: make(map[string]*CloneWrapper),
			snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
		}

		w := &CloneWrapper{Clone: &models.Clone{
			ID:        "fatal",
			Protected: true,
			Status:    models.Status{Code: models.StatusFatal},
			Snapshot:  &models.Snapshot{ID: "snap-1"},
		}}
		base.setWrapper("fatal", w)

		err := base.destroyPreChecks("fatal", w)
		// returns errNoSession because Session is nil, but crucially does NOT
		// return "clone is protected" -- the fatal status bypasses protection.
		assert.ErrorIs(t, err, errNoSession)
	})
}

func TestIsProtected(t *testing.T) {
	tests := []struct {
		name      string
		clone     models.Clone
		protected bool
	}{
		{name: "not protected", clone: models.Clone{Protected: false}, protected: false},
		{name: "protected with no expiry (forever)", clone: models.Clone{Protected: true}, protected: true},
		{name: "protected with future expiry", clone: models.Clone{Protected: true, ProtectedTill: &models.LocalTime{Time: time.Now().Add(time.Hour)}}, protected: true},
		{name: "protected but expired", clone: models.Clone{Protected: true, ProtectedTill: &models.LocalTime{Time: time.Now().Add(-time.Hour)}}, protected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.protected, tt.clone.IsProtected())
		})
	}
}

func TestProtectionExpiresIn(t *testing.T) {
	t.Run("not protected returns zero", func(t *testing.T) {
		c := models.Clone{Protected: false}
		assert.Equal(t, time.Duration(0), c.ProtectionExpiresIn())
	})

	t.Run("protected without expiry returns zero", func(t *testing.T) {
		c := models.Clone{Protected: true}
		assert.Equal(t, time.Duration(0), c.ProtectionExpiresIn())
	})

	t.Run("protected with future expiry returns positive duration", func(t *testing.T) {
		c := models.Clone{Protected: true, ProtectedTill: &models.LocalTime{Time: time.Now().Add(30 * time.Minute)}}
		d := c.ProtectionExpiresIn()
		assert.Greater(t, d, 29*time.Minute)
		assert.LessOrEqual(t, d, 31*time.Minute)
	})

	t.Run("protected with past expiry returns zero", func(t *testing.T) {
		c := models.Clone{Protected: true, ProtectedTill: &models.LocalTime{Time: time.Now().Add(-time.Hour)}}
		assert.Equal(t, time.Duration(0), c.ProtectionExpiresIn())
	})
}

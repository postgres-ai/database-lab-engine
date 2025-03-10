/*
2021 © Postgres.ai
*/

package cloning

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func (s *BaseCloningSuite) TestLatestSnapshot() {
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)

	snapshot1 := &models.Snapshot{
		ID:          "TestSnapshotID1",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 02, 20, 01, 23, 45, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 19, 0, 0, 0, 0, time.UTC)},
	}

	snapshot2 := &models.Snapshot{
		ID:          "TestSnapshotID2",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 02, 20, 05, 43, 21, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 20, 0, 0, 0, 0, time.UTC)},
	}

	require.Equal(s.T(), 0, len(s.cloning.snapshotBox.items))
	latestSnapshot, err := s.cloning.getLatestSnapshot()
	require.Nil(s.T(), latestSnapshot)
	require.EqualError(s.T(), err, "no snapshot found")

	s.cloning.addSnapshot(snapshot1)
	s.cloning.addSnapshot(snapshot2)

	latestSnapshot, err = s.cloning.getLatestSnapshot()
	require.NoError(s.T(), err)
	require.Equal(s.T(), latestSnapshot, snapshot2)

	snapshot3 := &models.Snapshot{
		ID:          "TestSnapshotID3",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 02, 21, 05, 43, 21, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 21, 0, 0, 0, 0, time.UTC)},
	}

	snapshotMap := make(map[string]*models.Snapshot)
	snapshotMap[snapshot1.ID] = snapshot1
	snapshotMap[snapshot2.ID] = snapshot2
	snapshotMap[snapshot3.ID] = snapshot3
	s.cloning.resetSnapshots(snapshotMap, snapshot3)

	require.Equal(s.T(), 3, len(s.cloning.snapshotBox.items))
	latestSnapshot, err = s.cloning.getLatestSnapshot()
	require.NoError(s.T(), err)
	require.Equal(s.T(), latestSnapshot, snapshot3)
}

func (s *BaseCloningSuite) TestSnapshotByID() {
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)

	snapshot1 := &models.Snapshot{
		ID:          "TestSnapshotID1",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 02, 20, 01, 23, 45, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 19, 0, 0, 0, 0, time.UTC)},
	}

	snapshot2 := &models.Snapshot{
		ID:          "TestSnapshotID2",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 02, 20, 05, 43, 21, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 20, 0, 0, 0, 0, time.UTC)},
	}

	require.Equal(s.T(), 0, len(s.cloning.snapshotBox.items))
	latestSnapshot, err := s.cloning.getLatestSnapshot()
	require.Nil(s.T(), latestSnapshot)
	require.EqualError(s.T(), err, "no snapshot found")

	s.cloning.addSnapshot(snapshot1)
	require.Equal(s.T(), 1, len(s.cloning.snapshotBox.items))
	require.Equal(s.T(), "TestSnapshotID1", s.cloning.snapshotBox.items[snapshot1.ID].ID)

	s.cloning.addSnapshot(snapshot2)
	require.Equal(s.T(), 2, len(s.cloning.snapshotBox.items))
	require.Equal(s.T(), "TestSnapshotID2", s.cloning.snapshotBox.items[snapshot2.ID].ID)

	latestSnapshot, err = s.cloning.getSnapshotByID("TestSnapshotID2")
	require.NoError(s.T(), err)
	require.Equal(s.T(), latestSnapshot, snapshot2)

	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)
	require.Equal(s.T(), 0, len(s.cloning.snapshotBox.items))
	latestSnapshot, err = s.cloning.getLatestSnapshot()
	require.Nil(s.T(), latestSnapshot)
	require.EqualError(s.T(), err, "no snapshot found")
}

func TestCloneCounter(t *testing.T) {
	c := &Base{}
	c.snapshotBox.items = make(map[string]*models.Snapshot)
	snapshot := &models.Snapshot{
		ID:        "testSnapshotID",
		NumClones: 0,
	}
	c.snapshotBox.items[snapshot.ID] = snapshot

	snapshot, err := c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	require.Equal(t, 0, snapshot.NumClones)

	c.IncrementCloneNumber("testSnapshotID")
	snapshot, err = c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	require.Equal(t, 1, snapshot.NumClones)

	c.decrementCloneNumber("testSnapshotID")
	snapshot, err = c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	require.Equal(t, 0, snapshot.NumClones)
}

func TestInitialCloneCounter(t *testing.T) {
	c := &Base{}
	c.clones = make(map[string]*CloneWrapper)

	snapshot := &models.Snapshot{
		ID: "testSnapshotID",
	}

	snapshot2 := &models.Snapshot{
		ID: "testSnapshotID2",
	}

	cloneWrapper01 := &CloneWrapper{
		Clone: &models.Clone{
			ID:       "test_clone001",
			Snapshot: snapshot,
		},
	}

	cloneWrapper02 := &CloneWrapper{
		Clone: &models.Clone{
			ID:       "test_clone002",
			Snapshot: snapshot,
		},
	}

	cloneWrapper03 := &CloneWrapper{
		Clone: &models.Clone{
			ID:       "test_clone003",
			Snapshot: snapshot2,
		},
	}

	c.clones["test_clone001"] = cloneWrapper01
	c.clones["test_clone002"] = cloneWrapper02
	c.clones["test_clone003"] = cloneWrapper03

	counters := c.counterClones()

	require.Len(t, counters, 2)
	require.Len(t, counters["testSnapshotID"], 2)
	require.Len(t, counters["testSnapshotID2"], 1)
	require.Len(t, counters["testSnapshotID3"], 0)
	require.ElementsMatch(t, []string{"test_clone001", "test_clone002"}, counters["testSnapshotID"])
}

func TestLatestSnapshots(t *testing.T) {
	baseSnapshot := &models.Snapshot{
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 19, 0, 0, 0, 0, time.UTC)},
	}
	newSnapshot := &models.Snapshot{
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 21, 0, 0, 0, 0, time.UTC)},
	}
	oldSnapshot := &models.Snapshot{
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 01, 0, 0, 0, 0, time.UTC)},
	}

	testCases := []struct {
		latest, challenger, result *models.Snapshot
	}{
		{
			latest:     baseSnapshot,
			challenger: newSnapshot,
			result:     newSnapshot,
		},
		{
			latest:     baseSnapshot,
			challenger: oldSnapshot,
			result:     baseSnapshot,
		},
		{
			latest:     nil,
			challenger: oldSnapshot,
			result:     oldSnapshot,
		},
		{
			latest:     &models.Snapshot{},
			challenger: oldSnapshot,
			result:     oldSnapshot,
		},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.result, defineLatestSnapshot(tc.latest, tc.challenger))
	}
}

/*
2021 © Postgres.ai
*/

package cloning

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func (s *BaseCloningSuite) TestLatestSnapshot() {
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)

	snapshot1 := &models.Snapshot{
		ID:          "TestSnapshotID1",
		CreatedAt:   "2020-02-20 01:23:45",
		DataStateAt: "2020-02-19 00:00:00",
	}

	snapshot2 := &models.Snapshot{
		ID:          "TestSnapshotID2",
		CreatedAt:   "2020-02-20 05:43:21",
		DataStateAt: "2020-02-20 00:00:00",
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
		CreatedAt:   "2020-02-21 05:43:21",
		DataStateAt: "2020-02-21 00:00:00",
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
		CreatedAt:   "2020-02-20 01:23:45",
		DataStateAt: "2020-02-19 00:00:00",
	}

	snapshot2 := &models.Snapshot{
		ID:          "TestSnapshotID2",
		CreatedAt:   "2020-02-20 05:43:21",
		DataStateAt: "2020-02-20 00:00:00",
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

	c.incrementCloneNumber("testSnapshotID")
	snapshot, err = c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	require.Equal(t, 1, snapshot.NumClones)

	c.decrementCloneNumber("testSnapshotID")
	snapshot, err = c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	require.Equal(t, 0, snapshot.NumClones)
}

func TestLatestSnapshots(t *testing.T) {
	baseSnapshot := &models.Snapshot{
		DataStateAt: "2020-02-19 00:00:00",
	}
	newSnapshot := &models.Snapshot{
		DataStateAt: "2020-02-21 00:00:00",
	}
	oldSnapshot := &models.Snapshot{
		DataStateAt: "2020-02-01 00:00:00",
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

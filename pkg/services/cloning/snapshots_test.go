/*
2021 © Postgres.ai
*/

package cloning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
)

func TestCloneCounter(t *testing.T) {
	c := &baseCloning{}
	snapshot := models.Snapshot{
		ID:        "testSnapshotID",
		NumClones: 0,
	}
	c.snapshots = append(c.snapshots, snapshot)

	snapshot, err := c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	assert.Equal(t, 0, snapshot.NumClones)

	c.incrementCloneNumber("testSnapshotID")
	snapshot, err = c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	assert.Equal(t, 1, snapshot.NumClones)

	c.decrementCloneNumber("testSnapshotID")
	snapshot, err = c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	assert.Equal(t, 0, snapshot.NumClones)
}

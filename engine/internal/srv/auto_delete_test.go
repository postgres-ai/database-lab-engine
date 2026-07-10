/*
2026 © Postgres.ai
*/

package srv

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/webhooks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestNextDeleteState(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	retention := time.Hour
	past := now.Add(-time.Minute)
	future := now.Add(time.Minute)

	testCases := []struct {
		name          string
		protected     bool
		hasDependents bool
		current       *time.Time
		wantDeleteAt  *time.Time
		wantShould    bool
	}{
		{name: "protected clears schedule", protected: true, current: &future, wantDeleteAt: nil, wantShould: false},
		{name: "protected with no schedule stays cleared", protected: true, current: nil, wantDeleteAt: nil, wantShould: false},
		{name: "dependents reset the clock", hasDependents: true, current: &future, wantDeleteAt: nil, wantShould: false},
		{name: "first seen unused schedules deletion", current: nil, wantDeleteAt: timePtr(now.Add(retention)), wantShould: false},
		{name: "schedule not reached keeps it", current: &future, wantDeleteAt: &future, wantShould: false},
		{name: "schedule reached triggers deletion", current: &past, wantDeleteAt: &past, wantShould: true},
		{name: "schedule exactly now triggers deletion", current: &now, wantDeleteAt: &now, wantShould: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, should := nextDeleteState(now, tc.protected, tc.hasDependents, tc.current, retention)
			assert.Equal(t, tc.wantShould, should)

			if tc.wantDeleteAt == nil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.True(t, tc.wantDeleteAt.Equal(*got), "expected %v, got %v", tc.wantDeleteAt, got)
		})
	}
}

func TestSnapshotIsLeaf(t *testing.T) {
	headID := "pool/branch/main@head"
	branchHeads := map[string]struct{}{headID: {}}

	testCases := []struct {
		name    string
		details models.SnapshotDetails
		want    bool
	}{
		{name: "true leaf", details: models.SnapshotDetails{ID: "pool/branch/dev@leaf"}, want: true},
		{name: "has child", details: models.SnapshotDetails{ID: "s", Child: []string{"c"}}, want: false},
		{name: "is fork root", details: models.SnapshotDetails{ID: "s", Root: []string{"dev"}}, want: false},
		{name: "is branch label", details: models.SnapshotDetails{ID: "s", Branch: []string{"dev"}}, want: false},
		{name: "has clones", details: models.SnapshotDetails{ID: "s", Clones: []string{"pool/branch/dev/clone/r0"}}, want: false},
		{name: "is branch head in set", details: models.SnapshotDetails{ID: headID}, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, snapshotIsLeaf(tc.details, branchHeads))
		})
	}
}

func TestBranchHasDependents(t *testing.T) {
	// dev lineage: s1 (fork point) -> s2 (intermediate commit) -> s3 (head). s2 carries a native
	// ZFS clone (pool/branch/dev/cl-dev/r0), the residual committed-snapshot dataset DestroyClone
	// leaves under the branch once its working clone is removed.
	newRepo := func() *models.Repo {
		return &models.Repo{
			Snapshots: map[string]models.SnapshotDetails{
				"s1": {ID: "s1", Parent: "-", Root: []string{"dev"}, Child: []string{"s2"}},
				"s2": {ID: "s2", Parent: "s1", Child: []string{"s3"}, Clones: []string{"pool/branch/dev/cl-dev/r0"}},
				"s3": {ID: "s3", Parent: "s2", Child: []string{}},
			},
			Branches: map[string]string{"dev": "s3"},
		}
	}

	t.Run("residual committed dataset does not block deletion", func(t *testing.T) {
		assert.False(t, branchHasDependents(newRepo(), "dev", "s3", map[string]struct{}{}))
	})

	t.Run("registered clone on a branch snapshot blocks deletion", func(t *testing.T) {
		assert.True(t, branchHasDependents(newRepo(), "dev", "s3", map[string]struct{}{"s2": {}}))
	})

	t.Run("child branch forked from a branch snapshot blocks deletion", func(t *testing.T) {
		repo := newRepo()
		repo.Snapshots["s2"] = models.SnapshotDetails{ID: "s2", Parent: "s1", Child: []string{"s3"}, Root: []string{"feature"}}
		repo.Branches["feature"] = "s4"
		assert.True(t, branchHasDependents(repo, "dev", "s3", map[string]struct{}{}))
	})
}

func TestBranchHeadSet(t *testing.T) {
	repo := &models.Repo{
		Branches: map[string]string{"main": "pool@m", "dev": "pool/branch/dev@d"},
	}

	heads := branchHeadSet(repo)
	require.Len(t, heads, 2)
	assert.Contains(t, heads, "pool@m")
	assert.Contains(t, heads, "pool/branch/dev@d")
}

func TestSkipAutoDelete(t *testing.T) {
	poolName := "pool/pg14"

	testCases := []struct {
		name       string
		snapshotID string
		want       bool
	}{
		{name: "automatic pool snapshot", snapshotID: "pool/pg14@snapshot_20240101120000", want: true},
		{name: "pre snapshot", snapshotID: "pool/pg14/branch/main/clone@20240101120000_pre", want: true},
		{name: "no separator", snapshotID: "pool/pg14-broken", want: true},
		{name: "branch commit", snapshotID: "pool/pg14/branch/dev@20240101120000", want: false},
		{name: "branch name containing _pre is not skipped", snapshotID: "pool/pg14/branch/feature_pre@20240101120000", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, skipAutoDelete(tc.snapshotID, poolName))
		})
	}
}

func TestDeleteAtChanged(t *testing.T) {
	a := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	b := a.Add(time.Hour)

	testCases := []struct {
		name    string
		current *time.Time
		next    *time.Time
		want    bool
	}{
		{name: "both nil", current: nil, next: nil, want: false},
		{name: "set from nil", current: nil, next: &a, want: true},
		{name: "cleared to nil", current: &a, next: nil, want: true},
		{name: "same value", current: &a, next: &a, want: false},
		{name: "different value", current: &a, next: &b, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, deleteAtChanged(tc.current, tc.next))
		})
	}
}

func TestParseDeleteAt(t *testing.T) {
	t.Run("empty value", func(t *testing.T) {
		assert.Nil(t, parseDeleteAt(""))
	})

	t.Run("valid RFC3339", func(t *testing.T) {
		ts := "2026-06-19T12:00:00Z"
		got := parseDeleteAt(ts)
		require.NotNil(t, got)
		assert.Equal(t, ts, got.UTC().Format(time.RFC3339))
	})

	t.Run("malformed value is dropped", func(t *testing.T) {
		assert.Nil(t, parseDeleteAt("not-a-time"))
	})
}

func TestDeletionBudget(t *testing.T) {
	t.Run("honors the per-tick cap", func(t *testing.T) {
		budget := deletionBudget{remaining: 2}

		require.True(t, budget.available())
		budget.consume()
		require.True(t, budget.available())
		budget.consume()

		assert.False(t, budget.available())
		assert.True(t, budget.capped)
	})

	t.Run("zero budget reports capped immediately", func(t *testing.T) {
		budget := deletionBudget{remaining: 0}
		assert.False(t, budget.available())
		assert.True(t, budget.capped)
	})
}

func TestRetentionHelpers(t *testing.T) {
	t.Run("retention disabled by default", func(t *testing.T) {
		s := &Server{}
		assert.False(t, s.retentionEnabled())
	})

	t.Run("retention enabled by snapshot window", func(t *testing.T) {
		s := &Server{}
		s.retention.UnusedSnapshotMinutes = 30
		assert.True(t, s.retentionEnabled())
	})

	t.Run("retention enabled by branch window", func(t *testing.T) {
		s := &Server{}
		s.retention.UnusedBranchMinutes = 30
		assert.True(t, s.retentionEnabled())
	})

	t.Run("interval falls back to the default", func(t *testing.T) {
		s := &Server{}
		assert.Equal(t, defaultRetentionInterval, s.retentionInterval())
	})

	t.Run("interval honors the configured value", func(t *testing.T) {
		s := &Server{}
		s.retention.CheckIntervalMinutes = 10
		assert.Equal(t, 10*time.Minute, s.retentionInterval())
	})

	t.Run("max deletions falls back to the default", func(t *testing.T) {
		s := &Server{}
		assert.Equal(t, defaultMaxDeletionsPerTick, s.maxDeletionsPerTick())
	})

	t.Run("max deletions honors the configured value", func(t *testing.T) {
		s := &Server{}
		s.retention.MaxDeletionsPerTick = 5
		assert.Equal(t, 5, s.maxDeletionsPerTick())
	})
}

func TestEmitWebhook(t *testing.T) {
	t.Run("delivers when the channel has space", func(t *testing.T) {
		s := &Server{webhookCh: make(chan webhooks.EventTyper, 1)}
		event := webhooks.BasicEvent{EventType: webhooks.SnapshotDeleteEvent, EntityID: "pool@snap"}

		s.emitWebhook(event)

		require.Len(t, s.webhookCh, 1)
		assert.Equal(t, event, <-s.webhookCh)
	})

	t.Run("drops without blocking when the channel is full", func(t *testing.T) {
		s := &Server{webhookCh: make(chan webhooks.EventTyper, 1)}
		first := webhooks.BasicEvent{EventType: webhooks.SnapshotDeleteEvent, EntityID: "first"}
		second := webhooks.BasicEvent{EventType: webhooks.BranchDeleteEvent, EntityID: "second"}

		s.emitWebhook(first)
		s.emitWebhook(second)

		require.Len(t, s.webhookCh, 1)
		assert.Equal(t, first, <-s.webhookCh, "the second event must be dropped, not overwrite the first")
	})
}

func TestMapKeys(t *testing.T) {
	t.Run("returns all keys", func(t *testing.T) {
		keys := mapKeys(map[string]struct{}{"a": {}, "b": {}, "c": {}})
		assert.ElementsMatch(t, []string{"a", "b", "c"}, keys)
	})

	t.Run("empty set returns empty slice", func(t *testing.T) {
		assert.Empty(t, mapKeys(map[string]struct{}{}))
	})
}

func timePtr(t time.Time) *time.Time {
	return &t
}

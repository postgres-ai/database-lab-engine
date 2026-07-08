/*
2026 © Postgres.ai
*/

package srv

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// protectionFSM is a pool.FSManager whose only meaningful methods are GetProtection and Pool. It
// embeds the interface so unimplemented methods are absent at compile time without 40+ stub
// methods; the helpers under test call GetProtection and Pool only.
type protectionFSM struct {
	pool.FSManager
	name       string
	protection thinclones.ProtectionProperties
	err        error
}

func (m protectionFSM) GetProtection(string) (thinclones.ProtectionProperties, error) {
	return m.protection, m.err
}

func (m protectionFSM) Pool() *resources.Pool {
	return &resources.Pool{Name: m.name}
}

func TestApplyProtectionUpdate(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }
	uintPtr := func(u uint) *uint { return &u }
	noop := func(string) error { return nil }

	t.Run("rejects an empty update", func(t *testing.T) {
		require.Error(t, applyProtectionUpdate(0, nil, nil, nil, noop, noop))
	})

	t.Run("rejects protection and scheduled deletion together", func(t *testing.T) {
		err := applyProtectionUpdate(0, boolPtr(true), nil, models.NewLocalTime(time.Now()), noop, noop)
		require.Error(t, err)
	})

	t.Run("protect forever clears delete_at", func(t *testing.T) {
		var till, deleteAt string
		deleteAtCalled := false

		err := applyProtectionUpdate(0, boolPtr(true), nil, nil,
			func(v string) error { till = v; return nil },
			func(v string) error { deleteAt = v; deleteAtCalled = true; return nil })

		require.NoError(t, err)
		assert.Equal(t, models.ProtectionForever, till)
		assert.True(t, deleteAtCalled)
		assert.Empty(t, deleteAt)
	})

	t.Run("protect with duration sets an RFC3339 timestamp", func(t *testing.T) {
		var till string

		err := applyProtectionUpdate(0, boolPtr(true), uintPtr(60), nil,
			func(v string) error { till = v; return nil }, noop)

		require.NoError(t, err)
		parsed, perr := time.Parse(time.RFC3339, till)
		require.NoError(t, perr)
		assert.WithinDuration(t, time.Now().Add(time.Hour), parsed, time.Minute)
	})

	t.Run("protect duration is capped by the configured max", func(t *testing.T) {
		var till string

		err := applyProtectionUpdate(30, boolPtr(true), uintPtr(600), nil,
			func(v string) error { till = v; return nil }, noop)

		require.NoError(t, err)
		parsed, _ := time.Parse(time.RFC3339, till)
		assert.WithinDuration(t, time.Now().Add(30*time.Minute), parsed, time.Minute)
	})

	t.Run("deleteAt clears protection", func(t *testing.T) {
		var till, deleteAt string
		tillCalled := false

		err := applyProtectionUpdate(0, nil, nil, models.NewLocalTime(time.Now().Add(time.Hour)),
			func(v string) error { till = v; tillCalled = true; return nil },
			func(v string) error { deleteAt = v; return nil })

		require.NoError(t, err)
		assert.True(t, tillCalled)
		assert.Empty(t, till)
		assert.NotEmpty(t, deleteAt)
	})

	t.Run("unprotect clears protection only and leaves delete_at untouched", func(t *testing.T) {
		var till string
		tillCalled, deleteAtCalled := false, false

		err := applyProtectionUpdate(0, boolPtr(false), nil, nil,
			func(v string) error { till = v; tillCalled = true; return nil },
			func(string) error { deleteAtCalled = true; return nil })

		require.NoError(t, err)
		assert.True(t, tillCalled)
		assert.Empty(t, till)
		assert.False(t, deleteAtCalled, "unprotect must not touch delete_at")
	})
}

func TestBranchProtectionWrite(t *testing.T) {
	datasets := []branchDatasetRef{
		{fsm: protectionFSM{name: "poolA"}, dataset: "poolA/branch/dev"},
		{fsm: protectionFSM{name: "poolB"}, dataset: "poolB/branch/dev"},
		{fsm: protectionFSM{name: "poolC"}, dataset: "poolC/branch/dev"},
	}

	t.Run("writes every pool on success", func(t *testing.T) {
		written := []string{}
		err := branchProtectionWrite(datasets, func(d branchDatasetRef) error {
			written = append(written, d.dataset)
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"poolA/branch/dev", "poolB/branch/dev", "poolC/branch/dev"}, written)
	})

	t.Run("attempts all pools and aggregates errors when one fails", func(t *testing.T) {
		attempts := 0
		err := branchProtectionWrite(datasets, func(d branchDatasetRef) error {
			attempts++
			if d.dataset == "poolB/branch/dev" {
				return errors.New("boom")
			}
			return nil
		})
		require.Error(t, err)
		assert.Equal(t, 3, attempts, "must not stop at the first failing pool")
		assert.Contains(t, err.Error(), "poolB")
	})
}

func TestReadBranchProtection(t *testing.T) {
	t.Run("reports protected when any pool is protected", func(t *testing.T) {
		datasets := []branchDatasetRef{
			{fsm: protectionFSM{name: "poolA"}, dataset: "poolA/branch/dev"},
			{fsm: protectionFSM{name: "poolB", protection: thinclones.ProtectionProperties{ProtectedTill: models.ProtectionForever}},
				dataset: "poolB/branch/dev"},
		}
		protected, _, _ := readBranchProtection(datasets)
		assert.True(t, protected)
	})

	t.Run("reports unprotected when no pool is protected", func(t *testing.T) {
		datasets := []branchDatasetRef{
			{fsm: protectionFSM{name: "poolA"}, dataset: "poolA/branch/dev"},
			{fsm: protectionFSM{name: "poolB"}, dataset: "poolB/branch/dev"},
		}
		protected, _, _ := readBranchProtection(datasets)
		assert.False(t, protected)
	})

	t.Run("reports unprotected for an empty dataset set", func(t *testing.T) {
		protected, till, deleteAt := readBranchProtection(nil)
		assert.False(t, protected)
		assert.Nil(t, till)
		assert.Nil(t, deleteAt)
	})
}

func TestEnsureNotProtected(t *testing.T) {
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)

	t.Run("allows deletion when not currently protected", func(t *testing.T) {
		for _, till := range []string{"", past, "garbage"} {
			fsm := protectionFSM{protection: thinclones.ProtectionProperties{ProtectedTill: till}}
			require.NoError(t, ensureNotProtected(fsm, "pool@snap", "snapshot", "pool@snap"))
		}
	})

	t.Run("refuses a snapshot protected forever", func(t *testing.T) {
		fsm := protectionFSM{protection: thinclones.ProtectionProperties{ProtectedTill: models.ProtectionForever}}
		err := ensureNotProtected(fsm, "pool@snap", "snapshot", "pool@snap")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "snapshot pool@snap is protected")
	})

	t.Run("refuses a branch protected until a future time", func(t *testing.T) {
		fsm := protectionFSM{protection: thinclones.ProtectionProperties{ProtectedTill: future}}
		err := ensureNotProtected(fsm, "pool/branch/dev", "branch", "dev")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch dev is protected")
	})

	t.Run("propagates a protection read error", func(t *testing.T) {
		fsm := protectionFSM{err: errors.New("zfs unavailable")}
		err := ensureNotProtected(fsm, "pool@snap", "snapshot", "pool@snap")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "zfs unavailable")
	})
}

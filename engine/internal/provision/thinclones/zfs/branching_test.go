package zfs

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestSetProtectedTill(t *testing.T) {
	t.Run("sets the value on the target", func(t *testing.T) {
		runner := newRecordingRunner()
		m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

		require.NoError(t, m.SetProtectedTill("2026-06-17T14:30:00Z", "pool@snap1"))
		assert.Equal(t, "2026-06-17T14:30:00Z", runner.props["pool@snap1:dle:protected_till"])
	})

	t.Run("empty value clears the property to -", func(t *testing.T) {
		runner := newRecordingRunner()
		m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

		require.NoError(t, m.SetProtectedTill("", "pool@snap1"))
		assert.Equal(t, "-", runner.props["pool@snap1:dle:protected_till"])
	})

	t.Run("targets a branch dataset", func(t *testing.T) {
		runner := newRecordingRunner()
		m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

		require.NoError(t, m.SetProtectedTill(models.ProtectionForever, "pool/branch/main"))
		assert.Equal(t, models.ProtectionForever, runner.props["pool/branch/main:dle:protected_till"])
	})
}

func TestSetDeleteAt(t *testing.T) {
	t.Run("sets the value on the target", func(t *testing.T) {
		runner := newRecordingRunner()
		m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

		require.NoError(t, m.SetDeleteAt("2026-06-18T00:00:00Z", "pool@snap1"))
		assert.Equal(t, "2026-06-18T00:00:00Z", runner.props["pool@snap1:dle:delete_at"])
	})

	t.Run("empty value clears the property to -", func(t *testing.T) {
		runner := newRecordingRunner()
		m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

		require.NoError(t, m.SetDeleteAt("", "pool@snap1"))
		assert.Equal(t, "-", runner.props["pool@snap1:dle:delete_at"])
	})
}

func TestGetProtection(t *testing.T) {
	runner := newRecordingRunner()
	m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

	t.Run("returns empty when nothing is set", func(t *testing.T) {
		props, err := m.GetProtection("pool@snap1")
		require.NoError(t, err)
		assert.Empty(t, props.ProtectedTill)
		assert.Empty(t, props.DeleteAt)
	})

	t.Run("returns locally set values in a single call", func(t *testing.T) {
		require.NoError(t, m.SetProtectedTill("2026-06-17T14:30:00Z", "pool@snap2"))
		require.NoError(t, m.SetDeleteAt("2026-06-18T00:00:00Z", "pool@snap2"))

		props, err := m.GetProtection("pool@snap2")
		require.NoError(t, err)
		assert.Equal(t, "2026-06-17T14:30:00Z", props.ProtectedTill)
		assert.Equal(t, "2026-06-18T00:00:00Z", props.DeleteAt)
	})
}

func TestListProtection(t *testing.T) {
	runner := newRecordingRunner()
	// the property list must precede the dataset, matching real `zfs get` syntax.
	cmd := "zfs get -H -o name,property,value -s local -t snapshot -r dle:protected_till,dle:delete_at pool"
	runner.outputs[cmd] = "pool@snap1\tdle:protected_till\t2026-06-17T14:30:00Z\n" +
		"pool@snap1\tdle:delete_at\t-\n" +
		"pool@snap2\tdle:delete_at\t2026-06-18T00:00:00Z\n"

	m := Manager{runner: runner, config: Config{Pool: resources.NewPool("pool")}}

	protection, err := m.ListProtection()
	require.NoError(t, err)
	require.Len(t, protection, 2)

	assert.Equal(t, "2026-06-17T14:30:00Z", protection["pool@snap1"].ProtectedTill)
	assert.Empty(t, protection["pool@snap1"].DeleteAt)
	assert.Equal(t, "2026-06-18T00:00:00Z", protection["pool@snap2"].DeleteAt)
	assert.Empty(t, protection["pool@snap2"].ProtectedTill)
	assert.Contains(t, runner.cmds, cmd, "ListProtection must place the property list before the dataset")
}

func TestDestroyBranchDataset(t *testing.T) {
	const branchDataset = "pool/branch/main"

	t.Run("recursively destroys the branch dataset and its residual commit datasets", func(t *testing.T) {
		runner := newRecordingRunner()
		m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

		require.NoError(t, m.DestroyBranchDataset(branchDataset))

		assert.Contains(t, runner.cmds, "zfs destroy -R "+branchDataset,
			"the branch dataset must be destroyed recursively so residual commit datasets are swept")
	})
}

func TestGetProtectedSnapshots(t *testing.T) {
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)

	runner := newRecordingRunner()
	cmd := "zfs get -H -o name,property,value -s local -t snapshot -r dle:protected_till,dle:delete_at pool"
	runner.outputs[cmd] = "pool@snap1\tdle:protected_till\t" + future + "\n" +
		"pool@snap2\tdle:protected_till\t" + past + "\n" +
		"pool@snap3\tdle:delete_at\t2026-06-18T00:00:00Z\n"

	m := Manager{runner: runner, config: Config{Pool: resources.NewPool("pool")}}

	protected, err := m.getProtectedSnapshots()
	require.NoError(t, err)
	assert.Equal(t, []string{"pool@snap1"}, protected)
}

func TestExcludeBusySnapshots(t *testing.T) {
	t.Run("empty list returns empty filter", func(t *testing.T) {
		assert.Equal(t, "", excludeBusySnapshots(nil))
	})

	t.Run("escapes regex metacharacters", func(t *testing.T) {
		out := excludeBusySnapshots([]string{"pool@snap.1", "pool@snap+2"})
		assert.True(t, strings.HasPrefix(out, "| grep -Ev '"))
		assert.Contains(t, out, `pool@snap\.1`)
		assert.Contains(t, out, `pool@snap\+2`)
	})
}

func TestProtectionTimestampRoundTrip(t *testing.T) {
	runner := newRecordingRunner()
	m := NewFSManager(runner, Config{Pool: resources.NewPool("pool")})

	original := time.Date(2026, 6, 17, 14, 30, 0, 0, time.UTC)
	stored := original.Format(time.RFC3339)

	require.NoError(t, m.SetProtectedTill(stored, "pool@snap1"))

	props, err := m.GetProtection("pool@snap1")
	require.NoError(t, err)

	parsed, err := time.Parse(time.RFC3339, props.ProtectedTill)
	require.NoError(t, err)
	assert.True(t, original.Equal(parsed), "round-trip must not corrupt the timestamp via the trailing - trim")
}

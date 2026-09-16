package observer

import (
	"context"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestMaskingField(t *testing.T) {
	testCases := []struct {
		logEntry       []string
		maskIndexes    []int
		expectedResult []string
	}{
		{
			logEntry:       []string{"explain select 5;", "select * from users where email = 'abc@example.com';"},
			maskIndexes:    []int{0},
			expectedResult: []string{"explain xxx;", "select * from users where email = 'abc@example.com';"},
		},
		{
			logEntry:       []string{"explain select 5;", "select * from users where email = 'abc@example.com';"},
			maskIndexes:    []int{1},
			expectedResult: []string{"explain select 5;", "select * from users where email = 'xxx@example.com';"},
		},
	}

	o := Observer{
		replacementRules: []ReplacementRule{
			{
				re:      regexp.MustCompile(`select (\d+)`),
				replace: "xxx",
			},
			{
				re:      regexp.MustCompile(`[a-z0-9._%+\-]+(@[a-z0-9.\-]+\.[a-z]{2,4})`),
				replace: "xxx$1",
			},
		},
	}

	for _, tc := range testCases {
		testLogEntry := make([]string, len(tc.logEntry))
		copy(testLogEntry, tc.logEntry)
		o.maskLogs(testLogEntry, tc.maskIndexes)
		assert.Equal(t, tc.expectedResult, testLogEntry)
	}
}

func newTestObserver() *Observer {
	return &Observer{
		sessionMu: &sync.Mutex{},
		storage:   make(map[string]*ObservingClone),
		cfg:       &Config{},
	}
}

func TestObserver_GetObservingClone(t *testing.T) {
	o := newTestObserver()
	clone := &ObservingClone{cloneID: "clone1"}
	o.storage["clone1"] = clone

	t.Run("existing clone", func(t *testing.T) {
		got, err := o.GetObservingClone("clone1")
		require.NoError(t, err)
		assert.Equal(t, clone, got)
	})

	t.Run("non-existing clone", func(t *testing.T) {
		got, err := o.GetObservingClone("unknown")
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "observer not found")
	})
}

func TestObserver_RemoveObservingClone(t *testing.T) {
	o := newTestObserver()
	o.storage["clone1"] = &ObservingClone{cloneID: "clone1"}
	o.storage["clone2"] = &ObservingClone{cloneID: "clone2"}

	t.Run("remove existing clone", func(t *testing.T) {
		o.RemoveObservingClone("clone1")
		_, err := o.GetObservingClone("clone1")
		require.Error(t, err)
		_, err = o.GetObservingClone("clone2")
		require.NoError(t, err)
	})

	t.Run("remove non-existing clone is no-op", func(t *testing.T) {
		o.RemoveObservingClone("unknown")
		assert.Len(t, o.storage, 1)
	})
}

func TestNewSession(t *testing.T) {
	startedAt := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	cfg := types.Config{ObservationInterval: 5, MaxLockDuration: 10, MaxDuration: 3600}
	tags := map[string]string{"env": "test", "team": "platform"}

	session := NewSession(42, startedAt, cfg, tags)

	require.NotNil(t, session)
	assert.Equal(t, uint64(42), session.SessionID)
	assert.Equal(t, startedAt, session.StartedAt)
	assert.Equal(t, cfg, session.Config)
	assert.Equal(t, tags, session.Tags)
	assert.True(t, session.FinishedAt.IsZero(), "finished_at should be zero for a new session")
}

func TestSession_IsFinished(t *testing.T) {
	testCases := []struct {
		name       string
		finishedAt time.Time
		expected   bool
	}{
		{name: "zero time means not finished", finishedAt: time.Time{}, expected: false},
		{name: "non-zero time means finished", finishedAt: time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC), expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := Session{FinishedAt: tc.finishedAt}
			assert.Equal(t, tc.expected, s.IsFinished())
		})
	}
}

func TestObservingClone_Session(t *testing.T) {
	t.Run("nil session returns nil", func(t *testing.T) {
		oc := &ObservingClone{}
		assert.Nil(t, oc.Session())
	})

	t.Run("returns a copy of session", func(t *testing.T) {
		original := &Session{SessionID: 7, StartedAt: time.Now()}
		oc := &ObservingClone{session: original}
		got := oc.Session()
		require.NotNil(t, got)
		assert.Equal(t, original.SessionID, got.SessionID)
		assert.NotSame(t, original, got, "should return a copy, not the same pointer")
	})
}

func TestObservingClone_ArtifactRegistry(t *testing.T) {
	oc := &ObservingClone{registryMu: &sync.Mutex{}, sessionRegistry: make(map[uint64]struct{})}

	t.Run("empty registry returns empty list", func(t *testing.T) {
		assert.Empty(t, oc.GetArtifactList())
		assert.False(t, oc.IsExistArtifacts(1))
	})

	t.Run("add and check artifact", func(t *testing.T) {
		oc.AddArtifact(10)
		oc.AddArtifact(20)
		assert.True(t, oc.IsExistArtifacts(10))
		assert.True(t, oc.IsExistArtifacts(20))
		assert.False(t, oc.IsExistArtifacts(30))
		assert.Len(t, oc.GetArtifactList(), 2)
	})
}

func TestObservingClone_ConfigAndCsvFields(t *testing.T) {
	cfg := types.Config{ObservationInterval: 15, MaxLockDuration: 30, MaxDuration: 7200}
	oc := NewObservingClone(cfg, nil)

	assert.Equal(t, cfg, oc.Config())
	assert.Contains(t, oc.CsvFields(), "log_time")
	assert.Contains(t, oc.CsvFields(), "application_name")
}

func TestNewObservingClone_DefaultConfig(t *testing.T) {
	t.Run("zero values get defaults", func(t *testing.T) {
		oc := NewObservingClone(types.Config{}, nil)
		assert.Equal(t, uint64(defaultIntervalSeconds), oc.config.ObservationInterval)
		assert.Equal(t, uint64(defaultMaxLockDurationSeconds), oc.config.MaxLockDuration)
		assert.Equal(t, uint64(defaultMaxDurationSeconds), oc.config.MaxDuration)
	})

	t.Run("custom values are preserved", func(t *testing.T) {
		cfg := types.Config{ObservationInterval: 5, MaxLockDuration: 20, MaxDuration: 300}
		oc := NewObservingClone(cfg, nil)
		assert.Equal(t, uint64(5), oc.config.ObservationInterval)
		assert.Equal(t, uint64(20), oc.config.MaxLockDuration)
		assert.Equal(t, uint64(300), oc.config.MaxDuration)
	})
}

func TestObservingClone_FillMaskedIndexes(t *testing.T) {
	oc := NewObservingClone(types.Config{}, nil)
	assert.NotEmpty(t, oc.maskedIndexes, "masked indexes should be populated from default csv fields")

	for _, idx := range oc.maskedIndexes {
		assert.GreaterOrEqual(t, idx, 0)
	}
}

func TestNewObserver(t *testing.T) {
	cfg := &Config{ReplacementRules: map[string]string{`\d+`: "NUM", `[a-z]+`: "WORD"}}
	o := NewObserver(nil, cfg, nil)

	require.NotNil(t, o)
	assert.NotNil(t, o.storage)
	assert.NotNil(t, o.sessionMu)
	assert.Len(t, o.replacementRules, 2)
}

func TestObservingClone_SetOverallError(t *testing.T) {
	session := &Session{}
	oc := &ObservingClone{session: session}

	oc.SetOverallError(true)
	assert.True(t, oc.session.state.OverallError)

	oc.SetOverallError(false)
	assert.False(t, oc.session.state.OverallError)
}

func TestObservingClone_RunSessionErrorSignalsDone(t *testing.T) {
	oc := NewObservingClone(types.Config{}, nil)

	require.Error(t, oc.RunSession(), "a session that was never initialized cannot run")

	select {
	case <-oc.done:
	case <-time.After(time.Second):
		t.Fatal("done must be signalled when RunSession exits with an error")
	}
}

func TestObservingClone_StopExpiredContext(t *testing.T) {
	oc := NewObservingClone(types.Config{}, nil)
	oc.session = &Session{SessionID: 42}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := oc.Stop(ctx)

	require.Error(t, err, "Stop must not block when RunSession never started")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestObservingClone_StopUninitializedSession(t *testing.T) {
	oc := NewObservingClone(types.Config{}, nil)

	require.Error(t, oc.Stop(context.Background()))
}

func TestObservingClone_StopTwiceAfterRunSessionExit(t *testing.T) {
	oc := NewObservingClone(types.Config{}, nil)
	oc.pool = &resources.Pool{Name: "pool", MountDir: t.TempDir()}
	oc.session = &Session{SessionID: 7, Config: types.Config{MaxDuration: 60}, Result: &models.ObservationResult{}}

	close(oc.done)

	require.NotPanics(t, func() {
		require.NoError(t, oc.Stop(context.Background()))
		require.NoError(t, oc.Stop(context.Background()))
	})
}

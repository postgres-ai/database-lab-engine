package cloning

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func BenchmarkGetClones(b *testing.B) {
	cloning := &Base{
		config:      &Config{},
		clones:      make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("clone-%03d", i)
		cloning.setWrapper(id, &CloneWrapper{
			Clone: &models.Clone{
				ID:        id,
				CreatedAt: &models.LocalTime{Time: time.Now()},
				Status:    models.Status{Code: models.StatusOK},
			},
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cloning.GetClones()
	}
}

func BenchmarkFindWrapper(b *testing.B) {
	cloning := &Base{
		clones:      make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("clone-%04d", i)
		cloning.setWrapper(id, &CloneWrapper{
			Clone: &models.Clone{ID: id},
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cloning.findWrapper("clone-0500")
	}
}

func BenchmarkGetSnapshotList(b *testing.B) {
	cloning := &Base{
		clones:      make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	for i := 0; i < 50; i++ {
		id := fmt.Sprintf("snap-%03d", i)
		cloning.snapshotBox.items[id] = &models.Snapshot{
			ID:        id,
			CreatedAt: &models.LocalTime{Time: time.Now()},
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cloning.getSnapshotList()
	}
}

func BenchmarkSetGetWrapper_Concurrent(b *testing.B) {
	cloning := &Base{
		clones:      make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			id := fmt.Sprintf("clone-%04d", i%100)
			cloning.setWrapper(id, &CloneWrapper{Clone: &models.Clone{ID: id}})
			cloning.findWrapper(id)
			i++
		}
	})
}

func BenchmarkGetCloningState(b *testing.B) {
	cloning := &Base{
		config:      &Config{},
		clones:      make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	for i := 0; i < 50; i++ {
		id := fmt.Sprintf("clone-%03d", i)
		cloning.setWrapper(id, &CloneWrapper{
			Clone: &models.Clone{
				ID:        id,
				CreatedAt: &models.LocalTime{Time: time.Now()},
				Status:    models.Status{Code: models.StatusOK},
			},
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cloning.GetCloningState()
	}
}

func TestBenchmarkSetup(t *testing.T) {
	cloning := &Base{
		config:      &Config{},
		clones:      make(map[string]*CloneWrapper),
		snapshotBox: SnapshotBox{items: make(map[string]*models.Snapshot)},
	}

	cloning.setWrapper("test", &CloneWrapper{Clone: &models.Clone{ID: "test"}})
	w, ok := cloning.findWrapper("test")
	require.True(t, ok)
	require.Equal(t, "test", w.Clone.ID)
}

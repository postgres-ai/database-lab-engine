package resources

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPool_DataStateAt(t *testing.T) {
	pool := NewPool("dblab_pool")
	assert.True(t, pool.DataStateAt().IsZero())

	dsa := time.Date(2026, 10, 8, 11, 55, 58, 0, time.UTC)
	pool.SetDSA(dsa)

	assert.Equal(t, dsa, pool.DataStateAt())
}

func TestPool_String(t *testing.T) {
	pool := &Pool{Name: "dblab_pool", Mode: "zfs", DSA: time.Date(2026, 10, 8, 11, 55, 58, 0, time.UTC), status: ActivePool}

	assert.Equal(t, "dblab_pool (mode: zfs, status: active, dataStateAt: 2026-10-08 11:55:58 +0000 UTC)", pool.String())
	assert.Equal(t, pool.String(), fmt.Sprint(pool))
}

func TestPool_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	pool := NewPool("dblab_pool")

	const iterations = 200

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			pool.SetDSA(time.Now())
			pool.SetStatus(RefreshingPool)
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			_ = pool.DataStateAt()
			_ = pool.Status()
			_ = fmt.Sprint(pool)
		}
	}()

	wg.Wait()
}

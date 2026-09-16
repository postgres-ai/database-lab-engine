package goroutine

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const waitTimeout = 5 * time.Second

func captureLog(t *testing.T) *syncBuffer {
	t.Helper()

	buf := &syncBuffer{}

	log.SetOutput(buf)
	t.Cleanup(log.ResetOutput)

	return buf
}

func TestRunRecoversPanic(t *testing.T) {
	buf := captureLog(t)

	panicked := Run("boom-worker", func() { panic("something broke") })

	assert.True(t, panicked)
	assert.Contains(t, buf.String(), `recovered panic in goroutine "boom-worker": something broke`)
	assert.Contains(t, buf.String(), "goroutine_test.go")
}

func TestRunWithoutPanic(t *testing.T) {
	buf := captureLog(t)

	called := false
	panicked := Run("quiet-worker", func() { called = true })

	assert.False(t, panicked)
	assert.True(t, called)
	assert.NotContains(t, buf.String(), "recovered panic")
}

func TestGoRecoversPanic(t *testing.T) {
	buf := captureLog(t)

	var wg sync.WaitGroup

	wg.Add(1)

	Go("go-worker", func() {
		defer wg.Done()

		panic("background failure")
	})

	wg.Wait()

	require.Eventually(t, func() bool {
		return strings.Contains(buf.String(), `recovered panic in goroutine "go-worker": background failure`)
	}, waitTimeout, 10*time.Millisecond)
}

func TestGoRunsFunction(t *testing.T) {
	done := make(chan struct{})

	Go("go-worker", func() { close(done) })

	select {
	case <-done:
	case <-time.After(waitTimeout):
		t.Fatal("function has not been called")
	}
}

func TestLoopRestartsAfterPanic(t *testing.T) {
	buf := captureLog(t)

	var runs atomic.Int32

	finished := make(chan struct{})

	go loop(context.Background(), "loop-worker", func() {
		if runs.Add(1) < 3 {
			panic("transient")
		}

		close(finished)
	}, time.Millisecond, 4*time.Millisecond)

	select {
	case <-finished:
	case <-time.After(waitTimeout):
		t.Fatal("loop has not recovered")
	}

	assert.Equal(t, int32(3), runs.Load())
	assert.Contains(t, buf.String(), `restarting goroutine "loop-worker" in 1ms`)
	assert.Contains(t, buf.String(), `restarting goroutine "loop-worker" in 2ms`)
}

func TestLoopResetsBackoffAfterStableRun(t *testing.T) {
	buf := captureLog(t)

	const maxDelay = 4 * time.Millisecond

	var runs atomic.Int32

	finished := make(chan struct{})

	go loop(context.Background(), "loop-worker", func() {
		switch runs.Add(1) {
		case 1, 2:
			panic("transient")
		case 3:
			time.Sleep(2 * maxDelay)
			panic("after a stable run")
		default:
			close(finished)
		}
	}, time.Millisecond, maxDelay)

	select {
	case <-finished:
	case <-time.After(waitTimeout):
		t.Fatal("loop has not recovered")
	}

	assert.Equal(t, int32(4), runs.Load())
	assert.Equal(t, 2, strings.Count(buf.String(), `restarting goroutine "loop-worker" in 1ms`))
	assert.Equal(t, 1, strings.Count(buf.String(), `restarting goroutine "loop-worker" in 2ms`))
}

func TestLoopStopsOnContextCancel(t *testing.T) {
	captureLog(t)

	ctx, cancel := context.WithCancel(context.Background())

	var runs atomic.Int32

	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		loop(ctx, "loop-worker", func() {
			runs.Add(1)
			cancel()
			panic("after cancel")
		}, time.Hour, time.Hour)
	}()

	select {
	case <-stopped:
	case <-time.After(waitTimeout):
		t.Fatal("loop has not stopped on context cancellation")
	}

	assert.Equal(t, int32(1), runs.Load())
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

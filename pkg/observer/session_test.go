package observer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSessionIsFinished(t *testing.T) {
	s := Session{}
	assert.False(t, s.IsFinished())

	s.FinishedAt = time.Now()
	assert.True(t, s.IsFinished())
}

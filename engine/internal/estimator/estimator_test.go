/*
2021 © Postgres.ai
*/

package estimator

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEstimateTiming(t *testing.T) {
	waitEvents := map[string]float64{
		"Running":                  45.63,
		"IO.DataFileRead":          17.60,
		"IO.WALSync":               17.00,
		"IO.DataFileImmediateSync": 10.97,
		"IO.BufFileRead":           2.21,
		"IO.BufFileWrite":          2.20,
		"IO.DataFileExtend":        2.20,
		"IO.WALWrite":              2.19,
	}

	const (
		readFactor   = 1.2
		writeFactor  = 1.2
		cloneTiming  = 9.53
		expectedTime = 7.09
	)

	est := NewTiming(waitEvents, readFactor, writeFactor)

	estimatedTime := est.CalcMin(cloneTiming)
	assert.Equal(t, expectedTime, math.Round(estimatedTime*100)/100)
}

func TestShouldEstimate(t *testing.T) {
	testCases := []struct {
		readRatio      float64
		writeRatio     float64
		shouldEstimate bool
	}{
		{
			readRatio:      0,
			writeRatio:     0,
			shouldEstimate: false,
		},
		{
			readRatio:      1,
			writeRatio:     1,
			shouldEstimate: false,
		},
		{
			readRatio:      1,
			writeRatio:     0,
			shouldEstimate: true,
		},
		{
			readRatio:      0,
			writeRatio:     1,
			shouldEstimate: true,
		},
		{
			readRatio:      0.5,
			writeRatio:     1.2,
			shouldEstimate: true,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.shouldEstimate, shouldEstimate(tc.readRatio, tc.writeRatio))
	}
}

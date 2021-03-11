/*
2021 © Postgres.ai
*/

package estimator

import (
	"fmt"
)

// delta defines insignificant difference between minimum and maximum values.
const delta = 0.05

// Timing defines a timing estimator.
type Timing struct {
	dbStat          *StatDatabase
	readPercentage  float64
	writePercentage float64
	normal          float64
	readRatio       float64
	writeRatio      float64
	readBlocks      uint64
}

// StatDatabase defines database blocks stats.
type StatDatabase struct {
	BlocksRead     int64   `json:"blks_read"`
	BlocksHit      int64   `json:"blks_hit"`
	BlockReadTime  float64 `json:"blk_read_time"`
	BlockWriteTime float64 `json:"blk_write_time"`
}

// NewTiming creates a new timing estimator.
func NewTiming(waitEvents map[string]float64, readRatio, writeRatio float64) *Timing {
	timing := &Timing{
		readRatio:  readRatio,
		writeRatio: writeRatio,
	}

	for event, percent := range waitEvents {
		switch {
		case isReadEvent(event):
			timing.readPercentage += percent

		case isWriteEvent(event):
			timing.writePercentage += percent

		default:
			timing.normal += percent
		}
	}

	return timing
}

// SetDBStat sets database stats.
func (est *Timing) SetDBStat(dbStat StatDatabase) {
	est.dbStat = &dbStat
}

// SetReadBlocks sets read blocks.
func (est *Timing) SetReadBlocks(readBlocks uint64) {
	est.readBlocks = readBlocks
}

// CalcMin calculates the minimum query time estimation for the production environment, given the prepared ratios.
func (est *Timing) CalcMin(elapsed float64) float64 {
	return (est.normal + est.writePercentage/est.writeRatio) / totalPercent * elapsed
}

// CalcMax calculates the maximum query time estimation for the production environment, given the prepared ratios.
func (est *Timing) CalcMax(elapsed float64) float64 {
	readPercentage := est.readPercentage

	if est.dbStat != nil && est.readBlocks != 0 {
		readSpeed := float64(est.dbStat.BlocksRead) / (est.dbStat.BlockReadTime / millisecondsInSecond)
		readPercentage = float64(est.readBlocks) / readSpeed
	}

	return (est.normal + readPercentage/est.readRatio + est.writePercentage/est.writeRatio) / totalPercent * elapsed
}

// EstTime prints estimation timings.
func (est *Timing) EstTime(elapsed float64) string {
	minTiming := est.CalcMin(elapsed)
	maxTiming := est.CalcMax(elapsed)

	estTime := fmt.Sprintf("%.3f...%.3f", minTiming, maxTiming)

	if maxTiming-minTiming <= delta {
		estTime = fmt.Sprintf("%.3f", maxTiming)
	}

	return fmt.Sprintf(" (estimated* for prod: %s s)", estTime)
}

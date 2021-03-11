/*
2021 © Postgres.ai
*/

// Package estimator provides tools to estimate query timing for a production environment.
package estimator

import (
	"context"
	"time"
)

// Config describes options to estimate query timing.
type Config struct {
	ReadRatio         float64       `yaml:"readRatio"`
	WriteRatio        float64       `yaml:"writeRatio"`
	ProfilingInterval time.Duration `yaml:"profilingInterval"`
	SampleThreshold   int           `yaml:"sampleThreshold"`
}

// Estimator defines a timing estimator.
type Estimator struct {
	cfg *Config
}

// NewEstimator creates a new Estimator.
func NewEstimator(cfg *Config) *Estimator {
	return &Estimator{cfg: cfg}
}

// Run starts profiling if it needs to be done.
func (e *Estimator) Run(ctx context.Context, p *Profiler) {
	if p.opts.SampleThreshold > 0 && shouldEstimate(e.cfg.ReadRatio, e.cfg.WriteRatio) {
		go p.Start(ctx)
		return
	}

	p.Stop()
}

// Reload reloads estimator configuration.
func (e *Estimator) Reload(cfg Config) {
	*e.cfg = cfg
}

// Config returns Estimator configuration.
func (e *Estimator) Config() Config {
	if e.cfg == nil {
		return Config{}
	}

	return *e.cfg
}

// shouldEstimate checks ratios to determine whether to skip an estimation.
func shouldEstimate(readRatio, writeRatio float64) bool {
	return (readRatio != 0 || writeRatio != 0) && (readRatio != 1 || writeRatio != 1)
}

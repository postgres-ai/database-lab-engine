/*
2020 © Postgres.ai
*/

// Package retrieval provides data retrieval pipeline.
package retrieval

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/components"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine"
)

// Retrieval describes a data retrieval.
type Retrieval struct {
	config       *config.Config
	stageBuilder components.StageBuilder
	stages       []components.StageRunner
}

// New creates a new data retrieval.
func New(cfg *dblabCfg.Config, dockerCLI *client.Client) (*Retrieval, error) {
	stageBuilder, err := engine.StageBuilder(&cfg.Global, dockerCLI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get stageBuilder")
	}

	return &Retrieval{
		config:       &cfg.Retrieval,
		stageBuilder: stageBuilder,
	}, nil
}

// Run start retrieving process.
func (p *Retrieval) Run(ctx context.Context) error {
	if err := p.parseStages(); err != nil {
		return errors.Wrap(err, "failed to parse retrieval stages")
	}

	for _, s := range p.stages {
		if err := s.Run(ctx); err != nil {
			return err
		}
	}

	return nil
}

// parseStages processes configuration to define data retrieval stages and jobs.
func (p *Retrieval) parseStages() error {
	for _, stageName := range p.config.Stages {
		stageRunner, err := p.stageBuilder.BuildStageRunner(stageName)
		if err != nil {
			return errors.Wrap(err, "failed to build stage")
		}

		for _, jobConfig := range p.config.StageSpec[stageName].Jobs {
			job, err := stageRunner.BuildJob(jobConfig)
			if err != nil {
				return errors.Wrap(err, "failed to build job")
			}

			stageRunner.AddJob(job)
		}

		p.addStage(stageRunner)
	}

	return nil
}

// addStage applies a stage to the current data retrieval.
func (p *Retrieval) addStage(stage components.StageRunner) {
	p.stages = append(p.stages, stage)
}

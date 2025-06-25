/*
2020 © Postgres.ai
*/

// Package query contains a query preprocessor.
package query

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const defaultWorkerCount = 2

// Processor represents a query preprocessor.
type Processor struct {
	docker   *client.Client
	dbName   string
	username string
	cfg      PreprocessorCfg
}

// PreprocessorCfg defines query preprocessing options.
type PreprocessorCfg struct {
	QueryPath  string `yaml:"queryPath"`
	MaxWorkers int    `yaml:"maxParallelWorkers"`
	Inline     string `yaml:"inline"`
}

// NewQueryProcessor creates a new query Processor.
func NewQueryProcessor(docker *client.Client, cfg PreprocessorCfg, dbName, username string) *Processor {
	if cfg.MaxWorkers == 0 {
		cfg.MaxWorkers = defaultWorkerCount
	}

	return &Processor{docker: docker, dbName: dbName, username: username, cfg: cfg}
}

// ApplyPreprocessingQueries applies queries to the Postgres instance inside the specified container.
func (q *Processor) ApplyPreprocessingQueries(ctx context.Context, containerID string) error {
	if q == nil {
		return nil
	}

	if err := q.applyFileQueries(ctx, containerID); err != nil {
		return fmt.Errorf("failed to apply preprocessing files from queryPath: %w", err)
	}

	if err := q.applyInlineQueries(ctx, containerID); err != nil {
		return fmt.Errorf("failed to apply preprocessing inline SQL: %w", err)
	}

	return nil
}

func (q *Processor) applyFileQueries(ctx context.Context, containerID string) error {
	if q.cfg.QueryPath == "" {
		return nil
	}

	infos, err := os.ReadDir(q.cfg.QueryPath)
	if err != nil {
		return err
	}

	for _, info := range infos {
		infoName := path.Join(q.cfg.QueryPath, info.Name())

		if info.IsDir() {
			if err := q.runParallel(ctx, containerID, infoName); err != nil {
				return err
			}

			continue
		}

		out, err := q.runSQLFile(ctx, containerID, infoName)
		if err != nil {
			return err
		}

		log.Msg(fmt.Sprintf("Run SQL: %s\n", infoName), out)
	}

	return nil
}

// runParallel executes queries simultaneously. Use a straightforward algorithm because the worker pool will be overhead.
func (q *Processor) runParallel(ctx context.Context, containerID, parallelDir string) error {
	infos, err := os.ReadDir(parallelDir)
	if err != nil {
		return err
	}

	tickets := make(chan struct{}, q.cfg.MaxWorkers)
	errCh := make(chan error, q.cfg.MaxWorkers)

	parallelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}

	log.Msg("Discovering preprocessing queries in the directory: ", parallelDir)

	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		select {
		case err = <-errCh:
			return err

		case tickets <- struct{}{}:
		}

		infoName := path.Join(parallelDir, info.Name())

		wg.Add(1)

		go func() {
			defer func() {
				wg.Done()
				<-tickets
			}()

			out, err := q.runSQLFile(parallelCtx, containerID, infoName)
			if err != nil {
				errCh <- err

				cancel()
				log.Err("preprocessing query: ", err)

				return
			}

			log.Msg(fmt.Sprintf("Result: %s\n", infoName), out)
		}()
	}

	wg.Wait()

	select {
	case err = <-errCh:
		return err
	default:
	}

	return nil
}

func (q *Processor) runSQLFile(ctx context.Context, containerID, filename string) (string, error) {
	psqlCommand := []string{"psql",
		"-U", q.username,
		"-d", q.dbName,
		"--file", filename,
	}

	log.Msg("Run psql command", psqlCommand)

	output, err := tools.ExecCommandWithOutput(ctx, q.docker, containerID, types.ExecConfig{Cmd: psqlCommand})

	return output, err
}

func (q *Processor) applyInlineQueries(ctx context.Context, containerID string) error {
	if q.cfg.Inline == "" {
		return nil
	}

	out, err := q.runInlineSQL(ctx, containerID, q.cfg.Inline)

	if err != nil {
		log.Dbg("Failed to execute inline SQL:", out)
		return err
	}

	log.Msg("Inline SQL has been successfully executed", out)

	return nil
}

func (q *Processor) runInlineSQL(ctx context.Context, containerID, inlineSQL string) (string, error) {
	psqlCommand := []string{"psql",
		"-U", q.username,
		"-d", q.dbName,
		"-c", inlineSQL,
	}

	log.Msg("Run psql command", psqlCommand)

	output, err := tools.ExecCommandWithOutput(ctx, q.docker, containerID, types.ExecConfig{Cmd: psqlCommand})

	return output, err
}

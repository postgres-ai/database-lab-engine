/*
2020 © Postgres.ai
*/

package snapshot

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

type queryProcessor struct {
	docker     *client.Client
	dbName     string
	username   string
	dirname    string
	maxWorkers int
}

func newQueryProcessor(docker *client.Client, dbName, username, scriptDir string, maxWorkers int) *queryProcessor {
	if maxWorkers == 0 {
		maxWorkers = defaultWorkerCount
	}

	return &queryProcessor{docker: docker, dbName: dbName, username: username, dirname: scriptDir, maxWorkers: maxWorkers}
}

func (q *queryProcessor) applyPreprocessingQueries(ctx context.Context, containerID string) error {
	infos, err := os.ReadDir(q.dirname)
	if err != nil {
		return err
	}

	for _, info := range infos {
		infoName := path.Join(q.dirname, info.Name())

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
func (q *queryProcessor) runParallel(ctx context.Context, containerID, parallelDir string) error {
	infos, err := os.ReadDir(parallelDir)
	if err != nil {
		return err
	}

	tickets := make(chan struct{}, q.maxWorkers)
	errCh := make(chan error, q.maxWorkers)

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
				log.Err("Preprocessing query: ", err)

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

func (q *queryProcessor) runSQLFile(ctx context.Context, containerID, filename string) (string, error) {
	psqlCommand := []string{"psql",
		"-U", q.username,
		"-d", q.dbName,
		"--file", filename,
	}

	log.Msg("Run psql command", psqlCommand)

	output, err := tools.ExecCommandWithOutput(ctx, q.docker, containerID, types.ExecConfig{Cmd: psqlCommand})

	return output, err
}

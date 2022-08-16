package logical

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v4"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/activity"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/db"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	dleRetrieval = "dle_retrieval"

	// queryFieldsNum defines the expected number of fields in the query result.
	queryFieldsNum = 5

	customRecordSeparator = "\\\\"

	// statQuery defines the query to get activity of the DLE retrieval.
	statQuery = `select 
  coalesce(usename, ''), 
  coalesce(extract(epoch from (clock_timestamp() - query_start)),0) as duration,
  left(regexp_replace(coalesce(query, ''), E'[ \\t\\r\\n]+', ' ', 'g'),100) as query_cut,
  coalesce(wait_event_type, ''), coalesce(wait_event, '')
  from pg_stat_activity
  where application_name = '` + dleRetrieval + "'"
)

func dbSourceActivity(ctx context.Context, dbCfg Connection) ([]activity.PGEvent, error) {
	connStr := db.ConnectionString(dbCfg.Host, strconv.Itoa(dbCfg.Port), dbCfg.Username, dbCfg.DBName, dbCfg.Password)

	querier, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %w", err)
	}

	rows, err := querier.Query(ctx, statQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to perform query to get DB activity: %w", err)
	}

	pgeList := make([]activity.PGEvent, 0)

	for rows.Next() {
		var pge activity.PGEvent
		if err := rows.Scan(&pge.User, &pge.Duration, &pge.Query, &pge.WaitEventType, &pge.WaitEvent); err != nil {
			return nil, fmt.Errorf("failed to scan the next row to the PG Activity result set: %w", err)
		}

		pgeList = append(pgeList, pge)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pgeList, nil
}

func pgContainerActivity(ctx context.Context, docker *client.Client, containerID string, db global.Database) ([]activity.PGEvent, error) {
	ins, err := docker.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check PG container activity: %w", err)
	}

	if ins.State.Health.Status != types.Healthy {
		log.Dbg("Target container is not ready yet: ", ins.State.Health.Status)
		return []activity.PGEvent{}, nil
	}

	activityCmd := []string{"psql", "-U", db.User(), "-d", db.Name(), "--record-separator='" + customRecordSeparator + "'", "-XAtc", statQuery}

	log.Msg("Running activity command: ", activityCmd)

	execCfg := types.ExecConfig{
		Tty: true,
		Cmd: activityCmd,
	}

	out, err := tools.ExecCommandWithOutput(ctx, docker, containerID, execCfg)

	if err != nil {
		log.Dbg("Activity command failed:", out)
		return nil, err
	}

	return parseStatActivity(out), nil
}

func parseStatActivity(queryResult string) []activity.PGEvent {
	activities := make([]activity.PGEvent, 0)

	// Cut out line breaks from the psql output and split records by the custom separator.
	lines := bytes.Split(bytes.ReplaceAll([]byte(queryResult), []byte("\r\n"), []byte("")), []byte(customRecordSeparator))

	for _, line := range lines {
		byteLine := bytes.TrimSpace(line)

		if len(byteLine) == 0 {
			continue
		}

		fields := bytes.Split(byteLine, []byte("|"))

		if fieldsLen := len(fields); fieldsLen != queryFieldsNum {
			log.Dbg(fmt.Sprintf("an invalid activity line given: %d fields are available, but requires %d",
				fieldsLen, queryFieldsNum), fields)
			continue
		}

		var queryDuration float64

		if durationString := string(fields[1]); durationString != "" {
			parsedDuration, err := strconv.ParseFloat(durationString, 64)
			if err != nil {
				log.Dbg("Cannot parse query duration:", durationString)
			}

			queryDuration = parsedDuration
		}

		activities = append(activities, activity.PGEvent{
			User:          string(fields[0]),
			Duration:      queryDuration,
			Query:         string(fields[2]),
			WaitEventType: string(fields[3]),
			WaitEvent:     string(fields[4]),
		})
	}

	return activities
}

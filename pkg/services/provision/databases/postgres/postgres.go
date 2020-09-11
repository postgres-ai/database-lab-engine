/*
2019-2020 © Postgres.ai
*/

// Package postgres provides an interface to work Postgres application.
package postgres

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq" // Register Postgres database driver.
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/databases/postgres/configuration"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/docker"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
)

const (
	// waitPostgresConnectionTimeout defines timeout to wait for Postgres initial connection.
	waitPostgresConnectionTimeout = 120

	// waitPostgresStartTimeout defines timeout to wait for Postgres start.
	waitPostgresStartTimeout = 360

	// checkPostgresStatusPeriod defines period to check Postgres status.
	checkPostgresStatusPeriod = 500

	// logsMinuteWindow defines number of minutes to get logs from container.
	logsMinuteWindow = 1
)

// Start starts Postgres instance.
func Start(r runners.Runner, c *resources.AppConfig) error {
	log.Dbg("Starting Postgres container...")

	if err := configuration.Run(c.DataDir()); err != nil {
		return errors.Wrap(err, "cannot update configs")
	}

	if _, err := docker.RunContainer(r, c); err != nil {
		return errors.Wrap(err, "failed to run container")
	}

	// Waiting for server to become ready and promote if needed.
	first := true
	cnt := 0
	waitPostgresTimeout := waitPostgresConnectionTimeout

	for {
		logs, err := docker.GetLogs(r, c, logsMinuteWindow)
		if err != nil {
			return errors.Wrap(err, "failed to read container logs")
		}

		fatalCount := strings.Count(logs, "FATAL")
		startingUpCount := strings.Count(logs, "FATAL: the database system is starting up")

		if fatalCount > 0 && startingUpCount != fatalCount {
			return errors.Wrap(fmt.Errorf("postgres fatal error"), "cannot start Postgres")
		}

		out, err := runSimpleSQL("select pg_is_in_recovery()", c)

		if err == nil {
			// Server does not need promotion if it is not in recovery.
			if out == "f" || out == "false" {
				break
			}

			// Run promotion if needed only first time.
			if out == "t" && first {
				log.Dbg("Postgres instance needs promotion.")

				// Increase Postgres start timeout for promotion.
				waitPostgresTimeout = waitPostgresStartTimeout

				first = false

				_, err = pgctlPromote(r, c)
				if err != nil {
					if runnerError := Stop(r, c); runnerError != nil {
						log.Err(runnerError)
					}

					return err
				}
			}
		} else {
			log.Err("Currently cannot connect to Postgres: ", out, err)
		}

		cnt++

		if cnt > waitPostgresTimeout {
			if runnerErr := Stop(r, c); runnerErr != nil {
				log.Err(runnerErr)
			}

			return errors.Wrap(err, "postgres start timeout")
		}

		time.Sleep(checkPostgresStatusPeriod * time.Millisecond)
	}

	return nil
}

// Stop stops Postgres instance.
func Stop(r runners.Runner, c *resources.AppConfig) error {
	log.Dbg("Stopping Postgres container...")

	if _, err := docker.RemoveContainer(r, c); err != nil {
		return errors.Wrap(err, "failed to remove container")
	}

	if _, err := r.Run("rm -rf " + c.UnixSocketCloneDir + "/*"); err != nil {
		return errors.Wrap(err, "failed to clean unix socket directory")
	}

	return nil
}

// List gets started Postgres instances filtered by label.
func List(r runners.Runner, label string) ([]string, error) {
	return docker.ListContainers(r, label)
}

func pgctlPromote(r runners.Runner, c *resources.AppConfig) (string, error) {
	promoteCmd := `pg_ctl --pgdata ` + c.DataDir() + ` ` +
		`-W ` + // No wait.
		`promote`

	return docker.Exec(r, c, promoteCmd)
}

// Generate postgres connection string.
func getPgConnStr(c *resources.AppConfig) string {
	var sb strings.Builder

	if c.Host != "" {
		sb.WriteString("host=" + c.Host + " ")
	}

	sb.WriteString("port=5432 ")

	if c.DBName() != "" {
		sb.WriteString("dbname=" + c.DBName() + " ")
	}

	if c.Username() != "" {
		sb.WriteString("user=" + c.Username() + " ")
	}

	if c.Password() != "" {
		sb.WriteString("password=" + c.Password() + " ")
	}

	return sb.String()
}

// Executes simple SQL commands which returns one string value.
func runSimpleSQL(command string, c *resources.AppConfig) (string, error) {
	connStr := getPgConnStr(c)
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		return "", errors.Wrap(err, "cannot connect to database")
	}

	result := ""
	row := db.QueryRow(command)
	err = row.Scan(&result)

	defer func() {
		err := db.Close()
		if err != nil {
			log.Err("Cannot close database connection.")
		}
	}()

	if err != nil && err == sql.ErrNoRows {
		return "", nil
	}

	return result, err
}

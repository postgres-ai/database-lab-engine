/*
2019-2020 © Postgres.ai
*/

// Package postgres provides an interface to work Postgres application.
package postgres

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq" // Register Postgres database driver.
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/databases/postgres/pgconfig"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/docker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
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

	if extraConf := c.ExtraConf(); len(extraConf) > 0 {
		configManager, err := pgconfig.NewCorrector(c.DataDir())
		if err != nil {
			return errors.Wrap(err, "failed to create a config manager")
		}

		if err := configManager.ApplyUserConfig(extraConf); err != nil {
			return errors.Wrap(err, "cannot apply user configs")
		}
	}

	if err := docker.RunContainer(r, c); err != nil {
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

		out, err := runSimpleSQL("select pg_is_in_recovery()", getPgConnStr(c.Host, c.DB.DBName, c.DB.Username, c.Port))

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
					if runnerError := Stop(r, c.Pool, c.CloneName); runnerError != nil {
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
			if runnerErr := Stop(r, c.Pool, c.CloneName); runnerErr != nil {
				log.Err(runnerErr)
			}

			return errors.Wrap(err, "postgres start timeout")
		}

		time.Sleep(checkPostgresStatusPeriod * time.Millisecond)
	}

	return nil
}

// Stop stops Postgres instance.
func Stop(r runners.Runner, p *resources.Pool, name string) error {
	log.Dbg("Stopping Postgres container...")

	if _, err := docker.RemoveContainer(r, name); err != nil {
		const errorPrefix = "Error: No such container:"

		if e, ok := err.(runners.RunnerError); ok && !strings.HasPrefix(e.Stderr, errorPrefix) {
			return errors.Wrap(err, "failed to remove container")
		}

		log.Msg("docker container was not found, ignore", err)
	}

	if _, err := r.Run("rm -rf " + p.SocketCloneDir(name) + "/*"); err != nil {
		return errors.Wrap(err, "failed to clean unix socket directory")
	}

	return nil
}

// List gets running Postgres instances filtered by label.
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
func getPgConnStr(host, dbname, username string, port uint) string {
	var sb strings.Builder

	if host != "" {
		sb.WriteString("host=" + host + " ")
	}

	sb.WriteString("port=" + strconv.Itoa(int(port)) + " ")
	sb.WriteString("dbname=" + dbname + " ")
	sb.WriteString("user=" + username + " ")

	return sb.String()
}

// runSimpleSQL executes simple SQL commands which returns one string value.
func runSimpleSQL(command, connStr string) (string, error) {
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

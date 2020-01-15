/*
2019-2020 © Postgres.ai
*/

package provision

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq" // Register Postgres database driver.
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

const (
	// modeImmediate defines an immediate mode.
	// We use pg_ctl -D ... -m immediate stop because we need to shut down
	// Postgres faster and completely get rid of this instance. So we don't care
	// about its state.
	modeImmediate = "immediate"

	// codeServerIsNotRunning defines a Postgres exit status code.
	// If the server is not running, the process returns an exit status of 3.
	codeServerIsNotRunning = 3

	// codeUnaccessibleDataDirectory defines a Postgres exit status code.
	// If an accessible data directory is not specified, the process returns an exit status of 4.
	codeUnaccessibleDataDirectory = 4

	// waitPostgtesTimeout defines timeout to wait Postgres ready.
	waitPostgtesTimeout = 360

	// checkPostgresStatusPeriod defines period to check Postgres status.
	checkPostgresStatusPeriod = 500
)

// PgConfig store Postgres configuration.
type PgConfig struct {
	Version string
	Bindir  string

	// PGDATA.
	Datadir string

	Host string
	Port uint
	Name string

	// The specified user must exist. The user will not be created automatically.
	Username string
	Password string
}

func (c PgConfig) getPortStr() string {
	return strconv.FormatUint(uint64(c.Port), 10)
}

func (c PgConfig) getLogsDir() string {
	return c.Datadir + string(os.PathSeparator) + "pg_log"
}

func (c PgConfig) getBindir() string {
	if len(c.Bindir) > 0 {
		return strings.TrimRight(c.Bindir, "/") + ""
	}

	// By default, we assume that we are working on Ubuntu/Debian.
	return fmt.Sprintf("/usr/lib/postgresql/%s/bin", c.Version)
}

func (c PgConfig) getUsername() string {
	if len(c.Username) > 0 {
		return c.Username
	}

	return "postgres"
}

func (c PgConfig) getPassword() string {
	if len(c.Password) > 0 {
		return c.Password
	}

	return "postgres"
}

func (c PgConfig) getDbName() string {
	if len(c.Name) > 0 {
		return c.Name
	}

	return "postgres"
}

// PostgresStart starts Postgres instance.
func PostgresStart(r Runner, c *PgConfig) error {
	log.Dbg("Starting Postgres...")

	logsDir := c.getLogsDir()

	createLogsCmd := "sudo -u postgres -s mkdir -p " + logsDir
	out, err := r.Run(createLogsCmd, true)

	if err != nil {
		return errors.Wrapf(err, "postgres start: make log dir %v", out)
	}

	// pg_ctl status mode checks whether a server is running
	// in the specified data directory.
	if _, err = pgctlStatus(r, c); err != nil {
		if runnerError, ok := err.(RunnerError); ok {
			switch runnerError.ExitStatus {
			case codeUnaccessibleDataDirectory:
				return errors.Wrap(runnerError, "cannot access PGDATA")

			case codeServerIsNotRunning:
				if _, err = pgctlStart(r, c); err != nil {
					return errors.Wrap(err, "failed to start via pgctl")
				}

			default:
				return errors.Wrap(runnerError, "an unknown runner error")
			}
		} else {
			return errors.Wrap(err, "an unknown error")
		}
	}
	// No errors – assume that the server is running.

	// Waiting for server to become ready and promoting if needed.
	first := true
	cnt := 0

	for {
		out, err := runSimpleSQL("select pg_is_in_recovery()", c)

		if err == nil {
			// Server does not need promotion if it is not in recovery.
			if out == "f" || out == "false" {
				break
			}

			// Run promotion if needed only first time.
			if out == "t" && first {
				log.Dbg("Postgres instance needs promotion.")

				first = false

				_, err = pgctlPromote(r, c)
				if err != nil {
					if runnerError := PostgresStop(r, c); runnerError != nil {
						log.Err(runnerError)
					}

					return err
				}
			}
		}

		cnt++

		if cnt > waitPostgtesTimeout { // 3 minutes
			if runnerErr := PostgresStop(r, c); runnerErr != nil {
				log.Err(runnerErr)
			}

			return errors.Wrap(err, "postgres could not be promoted within 3 minutes")
		}

		time.Sleep(checkPostgresStatusPeriod * time.Millisecond)
	}

	return nil
}

// PostgresStop stops Postgres instance.
func PostgresStop(r Runner, c *PgConfig) error {
	log.Dbg("Stopping Postgres...")

	var err error

	first := true
	cnt := 0

	for {
		// pg_ctl status mode checks whether a server is running
		// in the specified data directory.
		_, err = pgctlStatus(r, c)
		if err != nil {
			if runnerError, ok := err.(RunnerError); ok {
				switch runnerError.ExitStatus {
				case codeUnaccessibleDataDirectory:
					return errors.Wrap(runnerError, "cannot access PGDATA")

				case codeServerIsNotRunning:
					// Postgres stopped.
					return nil

				default:
					return errors.Wrap(runnerError, "an unknown runner error")
				}
			}

			return errors.Wrap(err, "an unknown error")
		}
		// No errors – assume that the server is running.

		if first {
			first = false

			if _, pgctlErr := pgctlStop(r, modeImmediate, c); pgctlErr != nil {
				return errors.Wrap(pgctlErr, "failed to stop via pgctl")
			}
		}

		cnt++

		if cnt > waitPostgtesTimeout { // 3 minutes
			return errors.Wrap(err, "postgres could not be stopped within 3 minutes")
		}

		time.Sleep(checkPostgresStatusPeriod * time.Millisecond)
	}
}

// PostgresList gets started Postgres instances.
func PostgresList(r Runner, prefix string) ([]string, error) {
	listProcsCmd := fmt.Sprintf(`ps ax`)

	out, err := r.Run(listProcsCmd, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list processes")
	}

	re := regexp.MustCompile(fmt.Sprintf(`(%s[0-9]+)`, prefix))

	return util.Unique(re.FindAllString(out, -1)), nil
}

func pgctlStart(r Runner, c *PgConfig) (string, error) {
	startCmd := `sudo --user postgres --non-interactive ` +
		c.getBindir() + `/pg_ctl ` +
		`--pgdata ` + c.Datadir + ` ` +
		`-o "-p ` + c.getPortStr() + `" ` +
		`-W ` + // No wait.
		`start`

	return r.Run(startCmd, true)
}

func pgctlStop(r Runner, mode string, c *PgConfig) (string, error) {
	stopCmd := `sudo --user postgres --non-interactive ` +
		c.getBindir() + `/pg_ctl ` +
		`--pgdata /` + c.Datadir + ` ` +
		`--mode ` + mode + ` ` +
		`-W ` + // No wait.
		`stop`

	return r.Run(stopCmd, true)
}

func pgctlStatus(r Runner, c *PgConfig) (string, error) {
	statusCmd := `sudo --user postgres --non-interactive ` +
		c.getBindir() + `/pg_ctl ` +
		`--pgdata ` + c.Datadir + ` ` +
		`status`

	return r.Run(statusCmd, true)
}

func pgctlPromote(r Runner, c *PgConfig) (string, error) {
	startCmd := `sudo --user postgres --non-interactive ` +
		c.getBindir() + `/pg_ctl ` +
		`--pgdata ` + c.Datadir + ` ` +
		`-W ` + // No wait.
		`promote`

	return r.Run(startCmd, true)
}

// Generate postgres connection string.
func getPgConnStr(c *PgConfig) string {
	var sb strings.Builder

	if len(c.Host) > 0 {
		sb.WriteString("host=" + c.Host + " ")
	}

	if len(c.getPortStr()) > 0 {
		sb.WriteString("port=" + c.getPortStr() + " ")
	}

	if len(c.getDbName()) > 0 {
		sb.WriteString("dbname=" + c.getDbName() + " ")
	}

	if len(c.getUsername()) > 0 {
		sb.WriteString("user=" + c.getUsername() + " ")
	}

	if len(c.getPassword()) > 0 {
		sb.WriteString(" password=" + c.getPassword() + "  ")
	}
	//sb.WriteString(" sslmode=disable")

	return sb.String()
}

// Executes simple SQL commands which returns one string value.
func runSimpleSQL(command string, c *PgConfig) (string, error) {
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

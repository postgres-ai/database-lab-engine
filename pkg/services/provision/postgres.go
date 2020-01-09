/*
2019 © Postgres.ai
*/

package provision

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

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
)

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

	LogsPrefix string
}

func (c PgConfig) getPortStr() string {
	return strconv.FormatUint(uint64(c.Port), 10)
}

func (c PgConfig) getLogsDir() string {
	portStr := c.getPortStr()
	prefix := c.LogsPrefix
	return prefix + "dblab_" + portStr + ".log"
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

func PostgresStart(r Runner, c *PgConfig) error {
	log.Dbg("Starting Postgres...")

	logsDir := c.getLogsDir()

	createLogsCmd := "sudo -u postgres -s touch " + logsDir
	out, err := r.Run(createLogsCmd, true)
	if err != nil {
		return errors.Wrapf(err, "postgres start: log touch %v", out)
	}

	// pg_ctl status mode checks whether a server is running
	// in the specified data directory.
	if _, err = pgctlStatus(r, c); err != nil {
		if runnerError, ok := err.(RunnerError); ok {
			switch runnerError.ExitStatus {
			case codeUnaccessibleDataDirectory:
				return errors.Wrap(runnerError, "cannot access PGDATA")

			case codeServerIsNotRunning:
				if _, err = pgctlStart(r, logsDir, c); err != nil {
					return errors.Wrap(err, "failed to start via pgctl")
				}

			default:
				return errors.Wrap(runnerError, "an unknown runner error")
			}
		}
		// TODO(akartasov): check non-RunnerError
	}
	// No errors – assume that the server is running.

	// Waiting for server to become ready and promoting if needed.
	first := true
	cnt := 0

	for {
		out, err = runPsql(r, "select pg_is_in_recovery()", c, false, false)

		if err == nil {
			// Server does not need promotion if it is not in recovery.
			if out == "f" {
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
		if cnt > 360 { // 3 minutes
			if runnerErr := PostgresStop(r, c); runnerErr != nil {
				log.Err(runnerErr)
			}

			return errors.Wrap(err, "postgres could not be promoted within 3 minutes")
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

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
			// TODO(akartasov): check non-RunnerError
		}
		// No errors – assume that the server is running.

		if first {
			first = false

			if _, pgctlErr := pgctlStop(r, modeImmediate, c); pgctlErr != nil {
				return errors.Wrap(pgctlErr, "failed to stop via pgctl")
			}
		}

		cnt++
		if cnt > 360 { // 3 minutes
			return errors.Wrap(err, "postgres could not be stopped within 3 minutes")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func PostgresList(r Runner, prefix string) ([]string, error) {
	listProcsCmd := fmt.Sprintf(`ps ax`)

	out, err := r.Run(listProcsCmd, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list processes")
	}

	re := regexp.MustCompile(fmt.Sprintf(`(%s[0-9]+)`, prefix))

	return util.Unique(re.FindAllString(out, -1)), nil
}

func pgctlStart(r Runner, logsDir string, c *PgConfig) (string, error) {
	startCmd := `sudo --user postgres --non-interactive ` +
		c.getBindir() + `/pg_ctl ` +
		`--pgdata ` + c.Datadir + ` ` +
		`--log ` + logsDir + ` ` +
		`-o "-p ` + c.getPortStr() + `" ` +
		`--no-wait ` +
		`start`

	return r.Run(startCmd, true)
}

func pgctlStop(r Runner, mode string, c *PgConfig) (string, error) {
	stopCmd := `sudo --user postgres --non-interactive ` +
		c.getBindir() + `/pg_ctl ` +
		`--pgdata /` + c.Datadir + ` ` +
		`--mode ` + mode + ` ` +
		`--no-wait ` +
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
		`--no-wait ` +
		`promote`

	return r.Run(startCmd, true)
}

// TODO(anatoly): Use SQL runner.
// Use `runPsqlStrict` for commands defined by a user!
func runPsql(r Runner, command string, c *PgConfig, formatted bool, useFile bool) (string, error) {
	host := ""
	if len(c.Host) > 0 {
		host = "--host " + c.Host + " "
	}

	params := "At" // Tuples only, unaligned.
	if formatted {
		params = ""
	}

	var filename string
	commandParam := fmt.Sprintf(`-c "%s"`, command)
	if useFile {
		source := rand.NewSource(time.Now().UnixNano())
		random := rand.New(source)
		uid := random.Uint64()

		filename := fmt.Sprintf("/tmp/psql-query-%d", uid)

		if err := ioutil.WriteFile(filename, []byte(command), 0644); err != nil {
			return "", errors.Wrap(err, "failed to write file")
		}

		commandParam = fmt.Sprintf(`-f %s`, filename)
	}

	psqlCmd := `PGPASSWORD=` + c.getPassword() + ` ` +
		c.getBindir() + `/psql ` +
		host +
		`--dbname ` + c.getDbName() + ` ` +
		`--port ` + c.getPortStr() + ` ` +
		`--username ` + c.getUsername() + ` ` +
		`-X` + params + ` ` +
		`--no-password ` +
		commandParam

	out, err := r.Run(psqlCmd)

	if useFile {
		_ = os.Remove(filename)
	}

	return out, errors.Wrap(err, "psql error")
}

// Use for user defined commands to DB. Currently we only need
// to support limited number of PSQL meta information commands.
// That's why it's ok to restrict usage of some symbols.
func runPsqlStrict(r Runner, command string, c *PgConfig) (string, error) {
	command = strings.Trim(command, " \n")
	if len(command) == 0 {
		return "", errors.New("empty command")
	}

	// Psql file option (-f) allows to run any number of commands.
	// We need to take measures to restrict multiple commands support,
	// as we only check the first command.

	// User can run backslash commands on the same line with the first
	// backslash command (even without space separator),
	// e.g. `\d table1\d table2`.

	// Remove all backslashes except the one in the beginning.
	command = string(command[0]) + strings.ReplaceAll(command[1:], "\\", "")

	// Semicolumn creates possibility to run consequent command.
	command = strings.ReplaceAll(command, ";", "")

	// User can run any command (including DML queries) on other lines.
	// Restricting usage of multiline commands.
	command = strings.ReplaceAll(command, "\n", "")

	out, err := runPsql(r, command, c, true, true)
	if err != nil {
		if rerr, ok := err.(RunnerError); ok {
			return "", errors.Wrapf(rerr, "runner pqsl error: %s", rerr.Stderr)
		}

		return "", errors.Wrap(err, "psql error")
	}

	return out, nil
}

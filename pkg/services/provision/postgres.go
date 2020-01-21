/*
2019-2020 © Postgres.ai
*/

package provision

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq" // Register Postgres database driver.
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

const (
	// waitPostgresTimeout defines timeout to wait Postgres ready.
	waitPostgresTimeout = 360

	// checkPostgresStatusPeriod defines period to check Postgres status.
	checkPostgresStatusPeriod = 500

	// pgCfgDir defines directory with Postgres configs.
	pgCfgDir = "postgres"

	// pgHbaConfName defines the name of HBA config.
	pgHbaConfName = "pg_hba.conf"

	// pgConfName defines the name of general Postgres config.
	pgConfName = "postgresql.conf"

	// logsMinuteWindow defines number of minutes to get logs from container.
	logsMinuteWindow = 1
)

// PgConfig store Postgres configuration.
type PgConfig struct {
	CloneName string

	Version     string
	DockerImage string

	// PGDATA.
	Datadir string

	UnixSocketCloneDir string

	Host string
	Port uint
	Name string

	// The specified user must exist. The user will not be created automatically.
	Username string
	Password string

	OSUsername string
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

	err := PostgresConfigure(c)
	if err != nil {
		return errors.Wrap(err, "cannot update configs")
	}

	_, err = DockerRunContainer(r, c)
	if err != nil {
		return errors.Wrap(err, "failed to run container")
	}

	// Waiting for server to become ready and promoting if needed.
	first := true
	cnt := 0

	for {
		logs, err := DockerGetLogs(r, c, logsMinuteWindow)
		if err != nil {
			return errors.Wrap(err, "failed to read container logs")
		}

		fatalCount := strings.Count(logs, "FATAL")
		startingUpCount := strings.Count(logs, "FATAL: the database system is starting up")

		if fatalCount > 0 && startingUpCount != fatalCount {
			return errors.Wrap(fmt.Errorf("postgres fatal error"), "cannot start Postgres")
		}

		out, err := runSimpleSQL("select pg_is_in_recovery()", c)

		log.Dbg("sql: out: ", out)
		log.Err("sql: err: ", err)

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

		if cnt > waitPostgresTimeout { // 3 minutes
			if runnerErr := PostgresStop(r, c); runnerErr != nil {
				log.Err(runnerErr)
			}

			return errors.Wrap(err, "postgres could not be promoted within 3 minutes")
		}

		time.Sleep(checkPostgresStatusPeriod * time.Millisecond)
	}

	return nil
}

// PostgresConfigure configures PGDATA with Database Lab configs.
func PostgresConfigure(c *PgConfig) error {
	log.Dbg("Configuring Postgres...")

	// Copy pg_hba.conf.
	pgHbaSrc, err := util.GetConfigPath(pgCfgDir + "/" + pgHbaConfName)
	if err != nil {
		return errors.Wrap(err, "cannot get path to pg_hba.conf in configs")
	}

	pgHbaDst := c.Datadir + "/" + pgHbaConfName

	input, err := ioutil.ReadFile(pgHbaSrc)
	if err != nil {
		return errors.Wrap(err, "cannot read "+pgHbaConfName+" from configs")
	}

	err = ioutil.WriteFile(pgHbaDst, input, 0644)
	if err != nil {
		return errors.Wrap(err, "cannot copy "+pgHbaConfName+" to PGDATA")
	}

	// Edit postgresql.conf.
	pgConfSrc, err := util.GetConfigPath(pgCfgDir + "/" + pgConfName)
	if err != nil {
		return errors.Wrap(err, "cannot get path to "+pgConfName+" in configs")
	}

	pgConfDst := c.Datadir + "/postgresql.conf"

	pgConfSrcFile, err := ioutil.ReadFile(pgConfSrc)
	if err != nil {
		return errors.Wrap(err, "cannot read "+pgConfName+" from configs")
	}

	pgConfDstFile, err := ioutil.ReadFile(pgConfDst)
	if err != nil {
		return errors.Wrap(err, "cannot read "+pgConfName+" from PGDATA")
	}

	pgConfSrcLines := strings.Split(string(pgConfSrcFile), "\n")
	pgConfDstLines := strings.Split(string(pgConfDstFile), "\n")

	for _, line := range pgConfSrcLines {
		if strings.HasPrefix(line, "##") {
			continue
		}

		// Comment lines.
		if strings.HasPrefix(line, "#") {
			param := strings.TrimSpace(strings.TrimPrefix(line, "#"))

			for i, lineDst := range pgConfDstLines {
				if strings.HasPrefix(lineDst, param) {
					pgConfDstLines[i] = "#" + lineDst
				}
			}

			continue
		}

		// Append lines.
		if len(strings.TrimSpace(line)) > 0 {
			pgConfDstLines = append(pgConfDstLines, line)
		}
	}

	output := strings.Join(pgConfDstLines, "\n")

	err = ioutil.WriteFile(pgConfDst, []byte(output), 0644)
	if err != nil {
		return errors.Wrap(err, "cannot write postgresql.conf to PGDATA")
	}

	return nil
}

// PostgresStop stops Postgres instance.
func PostgresStop(r Runner, c *PgConfig) error {
	log.Dbg("Stopping Postgres...")

	_, err := DockerStopContainer(r, c)
	if err != nil {
		return errors.Wrap(err, "failed to stop container")
	}

	_, err = DockerRemoveContainer(r, c)
	if err != nil {
		return errors.Wrap(err, "failed to remove container")
	}

	err = os.RemoveAll(c.UnixSocketCloneDir)
	if err != nil {
		return errors.Wrap(err, "failed to clean unix socket directory")
	}

	return nil
}

// PostgresList gets started Postgres instances.
func PostgresList(r Runner, prefix string) ([]string, error) {
	return DockerListContainers(r)
}

func pgctlPromote(r Runner, c *PgConfig) (string, error) {
	promoteCmd := `pg_ctl --pgdata ` + c.Datadir + ` ` +
		`-W ` + // No wait.
		`promote`

	return DockerExec(r, c, promoteCmd)
}

// Generate postgres connection string.
func getPgConnStr(c *PgConfig) string {
	var sb strings.Builder

	if len(c.Host) > 0 {
		sb.WriteString("host=" + c.Host + " ")
	}

	sb.WriteString("port=5432 ")

	if len(c.getDbName()) > 0 {
		sb.WriteString("dbname=" + c.getDbName() + " ")
	}

	if len(c.getUsername()) > 0 {
		sb.WriteString("user=" + c.getUsername() + " ")
	}

	if len(c.getPassword()) > 0 {
		sb.WriteString("password=" + c.getPassword() + " ")
	}

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

/*
2020 © Postgres.ai
*/

// Package configuration provides tools for work with Postgres configuration.
package configuration

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/fs"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

const (
	// pgCfgDir defines directory with Postgres configs.
	pgCfgDir = "postgres"

	// pgHbaConfName defines the name of HBA config.
	pgHbaConfName = "pg_hba.conf"

	// pgConfName defines the name of general Postgres config.
	pgConfName = "postgresql.conf"
)

// Corrector defines a struct to correct PostgreSQL configuration.
type Corrector struct {
	ExtraConfig map[string]string
}

// NewCorrector creates a new corrector.
func NewCorrector() *Corrector {
	return &Corrector{}
}

// NewCorrectorWithExtraConfig creates a new corrector with an extra configuration.
func NewCorrectorWithExtraConfig(extraConfig map[string]string) *Corrector {
	return &Corrector{ExtraConfig: extraConfig}
}

// Run corrects PGDATA with Database Lab configs.
func (c Corrector) Run(dataDir string) error {
	log.Dbg("Configuring Postgres...")

	// Copy pg_hba.conf.
	pgHbaSrc, err := util.GetConfigPath(path.Join(pgCfgDir, pgHbaConfName))
	if err != nil {
		return errors.Wrap(err, "cannot get path to pg_hba.conf in configs")
	}

	pgHbaDst := path.Join(dataDir, pgHbaConfName)

	input, err := ioutil.ReadFile(pgHbaSrc)
	if err != nil {
		return errors.Wrapf(err, "cannot read %s from configs", pgHbaConfName)
	}

	if err := ioutil.WriteFile(pgHbaDst, input, 0644); err != nil {
		return errors.Wrapf(err, "cannot copy %s to PGDATA", pgHbaConfName)
	}

	// Edit postgresql.conf.
	pgConfSrc, err := util.GetConfigPath(path.Join(pgCfgDir, pgConfName))
	if err != nil {
		return errors.Wrapf(err, "cannot get path to %s in configs", pgConfName)
	}

	pgConfDst := path.Join(dataDir, pgConfName)

	pgConfSrcFile, err := ioutil.ReadFile(pgConfSrc)
	if err != nil {
		return errors.Wrapf(err, "cannot read %s from configs", pgConfName)
	}

	pgConfDstFile, err := ioutil.ReadFile(pgConfDst)
	if err != nil {
		return errors.Wrapf(err, "cannot read %s from PGDATA", pgConfName)
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

	if err := ioutil.WriteFile(pgConfDst, []byte(output), 0644); err != nil {
		return errors.Wrap(err, "cannot write postgresql.conf to PGDATA")
	}

	return nil
}

// ApplyExtraConf applies extra configuration to the provided Postgres directory.
func (c Corrector) ApplyExtraConf(dataDir string) error {
	log.Dbg("Applying extra configuration")

	pgConf := path.Join(dataDir, pgConfName)
	pgConfLines := make([]string, 0, len(c.ExtraConfig))

	for configKey, configValue := range c.ExtraConfig {
		pgConfLines = append(pgConfLines, fmt.Sprintf("%s = '%s'", configKey, configValue))
	}

	output := strings.Join(pgConfLines, "\n")

	if err := fs.AppendFile(pgConf, []byte(output)); err != nil {
		return errors.Wrapf(err, "cannot write extra configuration to %s", pgConf)
	}

	return nil
}

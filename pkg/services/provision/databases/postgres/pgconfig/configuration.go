/*
2020 © Postgres.ai
*/

// Package pgconfig provides tools for work with Postgres configuration.
package pgconfig

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/defaults"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/tools/fs"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

const (
	initializedLabel = "## DBLAB_INITIALIZED"

	// defaultPgCfgDir defines directory with default Postgres configs.
	defaultPgCfgDir = "default"

	// pgCfgDir defines directory with Postgres configs.
	pgCfgDir = "postgres"

	// pgHbaConfName defines the name of HBA config.
	pgHbaConfName = "pg_hba.conf"

	// pgConfName defines the name of general Postgres config.
	pgConfName = "postgresql.conf"

	// recoveryConfName defines the name of recovery Postgres (<11) config.
	recoveryConfName = "recovery.conf"

	// Database Lab configuration files.
	// configPrefix defines a file prefix for Database Lab configuration files.
	configPrefix = "postgresql.dblab."

	// pgControlName describes a file to store significant pg_control configuration.
	pgControlName = "pg_control.conf"

	// syncConfigName describes a file to store sync configuration.
	syncConfigName = "sync.conf"

	// promotionConfigName describes a file to store promotion configuration.
	promotionConfigName = "promotion.conf"

	// snapshotConfigName describes a file to store snapshot configuration.
	snapshotConfigName = "snapshot.conf"

	// userConfigName declares a file to store user-defined configuration.
	userConfigName = "user_defined.conf"
)

var includedDBLabConfigFiles = []string{
	pgConfName,
	pgControlName,
	syncConfigName,
	promotionConfigName,
	snapshotConfigName,
	userConfigName,
}

// Manager defines a struct to correct PostgreSQL configuration.
type Manager struct {
	pgVersion float64
	dataDir   string
}

// NewCorrector creates a new corrector.
func NewCorrector(dataDir string) (*Manager, error) {
	m := &Manager{
		dataDir: dataDir,
	}

	if err := m.init(); err != nil {
		return nil, err
	}

	return m, nil
}

// GetPgVersion gets a version of Postgres Data.
func (m *Manager) GetPgVersion() float64 {
	return m.pgVersion
}

func (m *Manager) init() error {
	// TODO (akartasov): check initialized configs to skip this function.
	pgVersion, err := tools.DetectPGVersion(m.dataDir)
	if err != nil {
		return errors.Wrap(err, "failed to detect the Postgres version")
	}

	m.pgVersion = pgVersion

	// Add default configs to the Postgres directory.
	sourceConfigDir, err := util.GetConfigPath(path.Join(defaultPgCfgDir, fmt.Sprintf("%g", m.pgVersion)))
	if err != nil {
		return errors.Wrap(err, "cannot get path to default configs")
	}

	if err := fs.CopyDirectoryContent(sourceConfigDir, m.dataDir); err != nil {
		return errors.Wrap(err, "failed to set default configuration files")
	}

	// Include Database Lab components to the default Postgres configuration file.
	if err := m.rewritePostgresConfig(); err != nil {
		return errors.Wrap(err, "failed to rewrite config")
	}

	// Correct PGDATA with Database Lab configs.
	if err := m.adjustHBAConf(); err != nil {
		return errors.Wrap(err, "failed to adjust pg_hba PostgreSQL configs")
	}

	if err := m.adjustGeneralConfigs(); err != nil {
		return errors.Wrap(err, "failed to adjust general PostgreSQL configs")
	}

	return nil
}

// rewritePostgresConfig completely rewrites a default Postgres configuration file.
func (m *Manager) rewritePostgresConfig() error {
	pgConfDst := path.Join(m.dataDir, pgConfName)

	buf := bytes.NewBuffer([]byte{})
	buf.WriteString(initializedLabel + "\n")

	for _, configFile := range includedDBLabConfigFiles {
		if _, err := buf.WriteString(fmt.Sprintf("include_if_exists %s%s\n", configPrefix, configFile)); err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(pgConfDst, buf.Bytes(), 0644); err != nil {
		return errors.Wrapf(err, "cannot rewrite %s at PGDATA", pgConfDst)
	}

	return nil
}

// adjustHBAConf corrects pg_hba.conf with Database Lab configs.
func (m *Manager) adjustHBAConf() error {
	log.Dbg("Configuring pg_hba.conf...")

	// Copy pg_hba.conf.
	pgHbaSrc, err := util.GetConfigPath(path.Join(pgCfgDir, pgHbaConfName))
	if err != nil {
		return errors.Wrap(err, "cannot get path to pg_hba.conf in configs")
	}

	pgHbaDst := path.Join(m.dataDir, pgHbaConfName)

	input, err := ioutil.ReadFile(pgHbaSrc)
	if err != nil {
		return errors.Wrapf(err, "cannot read %s from configs", pgHbaConfName)
	}

	if err := ioutil.WriteFile(pgHbaDst, input, 0644); err != nil {
		return errors.Wrapf(err, "cannot copy %s to PGDATA", pgHbaConfName)
	}

	return nil
}

// adjustGeneralConfigs corrects general PostgreSQL parameters with Database Lab configs.
func (m Manager) adjustGeneralConfigs() error {
	log.Dbg("Configuring Postgres...")

	pgConfSrc, err := util.GetConfigPath(path.Join(pgCfgDir, pgConfName))
	if err != nil {
		return errors.Wrapf(err, "cannot get path to %s in configs", pgConfName)
	}

	pgConfDst := path.Join(m.dataDir, configPrefix+pgConfName)

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

	if len(pgConfDstLines) > 0 && pgConfDstLines[0] == initializedLabel {
		// Already enforced.
		return nil
	}

	pgConfDstLines = append(pgConfDstLines, initializedLabel)

	// Prepend enforced mark.
	if len(pgConfDstLines) > 1 {
		copy(pgConfDstLines[1:], pgConfDstLines)
		pgConfDstLines[0] = initializedLabel
	}

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

// AdjustRecoveryFiles adjusts a recovery files.
func (m *Manager) AdjustRecoveryFiles() error {
	if err := os.Remove(path.Join(m.dataDir, "postmaster.pid")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Wrap(err, "failed to remove postmaster.pid")
	}

	// Truncate pg_ident.conf.
	if err := tools.TouchFile(path.Join(m.dataDir, "pg_ident.conf")); err != nil {
		return errors.Wrap(err, "failed to truncate pg_ident.conf")
	}

	if m.pgVersion >= defaults.PGVersion12 {
		if err := tools.TouchFile(path.Join(m.dataDir, "standby.signal")); err != nil {
			return err
		}
	}

	return nil
}

// ApplyRecovery applies recovery configuration parameters.
func (m *Manager) ApplyRecovery(cfg map[string]string) error {
	if err := appendExtraConf(path.Join(m.dataDir, m.recoveryFilename()), cfg); err != nil {
		return err
	}

	return nil
}

// ApplyPgControl applies significant configuration parameters extracted by the pg_control tool.
func (m *Manager) ApplyPgControl(pgControl map[string]string) error {
	// TODO (akartasov): add a label check to skip an already initialized pg_control config.
	if err := m.rewriteConfig(m.getConfigPath(pgControlName), pgControl); err != nil {
		return err
	}

	return nil
}

// ApplySync applies configuration parameters for sync instance.
func (m *Manager) ApplySync(cfg map[string]string) error {
	if err := m.rewriteConfig(m.getConfigPath(syncConfigName), cfg); err != nil {
		return err
	}

	return nil
}

// TruncateSyncConfig truncates a sync configuration file.
func (m *Manager) TruncateSyncConfig() error {
	return m.truncateConfig(m.getConfigPath(promotionConfigName))
}

// ApplyPromotion applies promotion configuration parameters.
func (m *Manager) ApplyPromotion(cfg map[string]string) error {
	if err := m.rewriteConfig(m.getConfigPath(promotionConfigName), cfg); err != nil {
		return err
	}

	return nil
}

// TruncatePromotionConfig truncates a promotion configuration file.
func (m *Manager) TruncatePromotionConfig() error {
	return m.truncateConfig(m.getConfigPath(promotionConfigName))
}

// ApplySnapshot applies snapshot configuration parameters.
func (m *Manager) ApplySnapshot(cfg map[string]string) error {
	if err := m.rewriteConfig(m.getConfigPath(snapshotConfigName), cfg); err != nil {
		return err
	}

	return nil
}

// ApplyUserConfig applies user-defined configuration.
func (m *Manager) ApplyUserConfig(cfg map[string]string) error {
	if err := m.rewriteConfig(m.getConfigPath(userConfigName), cfg); err != nil {
		return err
	}

	return nil
}

// getConfigPath builds a path of the Database Lab config file.
func (m *Manager) getConfigPath(configName string) string {
	return path.Join(m.dataDir, configPrefix+configName)
}

// recoveryFilename returns the name of a recovery configuration file.
func (m Manager) recoveryFilename() string {
	if m.pgVersion > defaults.PGVersion12 {
		return pgConfName
	}

	return recoveryConfName
}

// rewriteConfig completely rewrite a configuration file with provided parameters.
func (m *Manager) rewriteConfig(pgConf string, extraConfig map[string]string) error {
	log.Dbg("Applying extra configuration")

	pgConfLines := make([]string, 0, len(extraConfig))

	for configKey, configValue := range extraConfig {
		pgConfLines = append(pgConfLines, fmt.Sprintf("%s = '%s'", configKey, configValue))
	}

	output := strings.Join(pgConfLines, "\n")

	if err := ioutil.WriteFile(pgConf, []byte(output), 0644); err != nil {
		return errors.Wrapf(err, "cannot write extra configuration to %s", pgConf)
	}

	return nil
}

// appendExtraConf appends extra parameters to a provided Postgres configuration file.
func appendExtraConf(pgConf string, extraConfig map[string]string) error {
	log.Dbg("Appending extra configuration")

	pgConfLines := make([]string, 0, len(extraConfig))

	for configKey, configValue := range extraConfig {
		pgConfLines = append(pgConfLines, fmt.Sprintf("%s = '%s'", configKey, configValue))
	}

	output := "\n" + strings.Join(pgConfLines, "\n")

	if err := fs.AppendFile(pgConf, []byte(output)); err != nil {
		return errors.Wrapf(err, "cannot write extra configuration to %s", pgConf)
	}

	return nil
}

// truncateConfig truncates a configuration file.
func (m *Manager) truncateConfig(pgConf string) error {
	return ioutil.WriteFile(pgConf, []byte{}, 0644)
}

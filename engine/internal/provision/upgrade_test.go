/*
2026 © Postgres.ai
*/

package provision

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
)

type stubRunner struct {
	out     string
	err     error
	lastCmd string
}

func (r *stubRunner) Run(command string, _ ...bool) (string, error) {
	r.lastCmd = command

	return r.out, r.err
}

func TestValidateUpgradeTarget(t *testing.T) {
	tests := []struct {
		name    string
		current float64
		target  int
		wantErr string
	}{
		{name: "one major up", current: 16, target: 17},
		{name: "maximum supported jump", current: 14, target: 18},
		{name: "same version", current: 16, target: 16, wantErr: "must be greater than the current version"},
		{name: "downgrade", current: 17, target: 16, wantErr: "must be greater than the current version"},
		{name: "jump beyond image coverage", current: 13, target: 18, wantErr: "at most 4 preceding majors"},
		{name: "oldest supported source", current: 12, target: 13},
		{name: "source older than the image carries", current: 11, target: 13, wantErr: "carries binaries for 12 and newer"},
		{name: "legacy dotted source names its real version", current: 9.6, target: 13, wantErr: "from PostgreSQL 9.6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpgradeTarget(tt.current, tt.target)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestClassifyOutcome(t *testing.T) {
	tests := []struct {
		name     string
		facts    dataDirFacts
		expected recoveryAction
	}{
		{name: "swap completed", facts: dataDirFacts{SwapCompleted: true}, expected: recoveryComplete},
		{name: "swap wins over a disabled old cluster", facts: dataDirFacts{SwapCompleted: true, OldClusterDisabled: true}, expected: recoveryComplete},
		{name: "converted and recorded but not yet promoted is an unfinished success", facts: dataDirFacts{NewClusterReady: true, OldClusterDisabled: true, ConversionDone: true}, expected: recoveryComplete},
		{name: "nothing converted", facts: dataDirFacts{}, expected: recoveryRollback},
		{name: "an initdb'd target alone never promotes the empty cluster", facts: dataDirFacts{NewClusterReady: true}, expected: recoveryRollback},
		{name: "old cluster disabled with nothing to promote", facts: dataDirFacts{OldClusterDisabled: true}, expected: recoveryReprovision},
		// pg_upgrade disables the old cluster when linking STARTS, so an interrupted transfer
		// presents an initdb'd target and an unstartable source and is indistinguishable from a
		// finished run by those two facts alone. Promoting here would publish a partially linked
		// cluster as a successful upgrade and then delete the only copy of the unlinked files.
		{name: "an interrupted transfer must never be promoted", facts: dataDirFacts{NewClusterReady: true, OldClusterDisabled: true}, expected: recoveryReprovision},
		{name: "a completion marker alone does not promote a startable old cluster", facts: dataDirFacts{NewClusterReady: true, ConversionDone: true}, expected: recoveryRollback},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, classifyOutcome(tt.facts))
		})
	}
}

func TestReadDataDirFacts(t *testing.T) {
	// markIn selects which cluster directory gets the completion marker: "data" for a cluster the
	// swap has already promoted, "new" for one still awaiting it, "" for none.
	newFactsMarked := func(t *testing.T, pgVersion string, controlPresent bool, newDirVersion, markIn string) dataDirFacts {
		t.Helper()

		appConfig := testAppConfig(t)

		require.NoError(t, os.MkdirAll(path.Join(appConfig.DataDir(), "global"), 0755))
		require.NoError(t, os.WriteFile(path.Join(appConfig.DataDir(), "PG_VERSION"), []byte(pgVersion), 0644))

		if controlPresent {
			require.NoError(t, os.WriteFile(path.Join(appConfig.DataDir(), "global", "pg_control"), []byte("x"), 0644))
		}

		if newDirVersion != "" {
			require.NoError(t, os.MkdirAll(newDataDir(appConfig), 0700))
			require.NoError(t, os.WriteFile(path.Join(newDataDir(appConfig), "PG_VERSION"), []byte(newDirVersion), 0644))
		}

		markerDirs := map[string]string{"data": appConfig.DataDir(), "new": newDataDir(appConfig)}
		if markerDir, ok := markerDirs[markIn]; ok {
			require.NoError(t, os.MkdirAll(markerDir, 0700))
			require.NoError(t, os.WriteFile(path.Join(markerDir, upgradeDoneMarkerName), nil, 0644))
		}

		return readDataDirFacts(appConfig, UpgradeState{OldVersion: "16", NewVersion: "17"})
	}

	newFacts := func(t *testing.T, pgVersion string, controlPresent bool, newDirVersion string) dataDirFacts {
		t.Helper()

		return newFactsMarked(t, pgVersion, controlPresent, newDirVersion, "")
	}

	t.Run("untouched old cluster", func(t *testing.T) {
		facts := newFacts(t, "16", true, "")
		assert.False(t, facts.SwapCompleted)
		assert.False(t, facts.OldClusterDisabled)
	})

	t.Run("pg_upgrade renamed pg_control away", func(t *testing.T) {
		facts := newFacts(t, "16", false, "")
		assert.False(t, facts.SwapCompleted)
		assert.True(t, facts.OldClusterDisabled, "a missing pg_control is what makes the old cluster unstartable")
	})

	t.Run("swap already happened", func(t *testing.T) {
		facts := newFacts(t, "17", true, "")
		assert.True(t, facts.SwapCompleted)
	})

	t.Run("an interrupted transfer records no completion", func(t *testing.T) {
		facts := newFacts(t, "16", false, "17")

		assert.False(t, facts.SwapCompleted)
		assert.True(t, facts.NewClusterReady, "initdb ran, which says nothing about the transfer")
		assert.True(t, facts.OldClusterDisabled, "pg_upgrade disables the old cluster before it moves files")
		assert.False(t, facts.ConversionDone, "no marker means the transfer never reported success")
	})

	t.Run("converted but not yet promoted", func(t *testing.T) {
		facts := newFactsMarked(t, "16", false, "17", "new")

		assert.True(t, facts.NewClusterReady)
		assert.True(t, facts.OldClusterDisabled)
		assert.True(t, facts.ConversionDone, "the marker in the new cluster is what proves the conversion finished")
	})

	// The data directory belongs to the clone's postgres user and survives between attempts, so a
	// marker there may be a leftover or may have been written through the running clone. It must
	// never count: were it to, an interrupted transfer - which presents NewClusterReady and
	// OldClusterDisabled on its own - would classify as a finished upgrade and be promoted.
	t.Run("a marker in the old cluster is never taken as completion", func(t *testing.T) {
		facts := newFactsMarked(t, "16", false, "17", "data")

		assert.True(t, facts.NewClusterReady)
		assert.True(t, facts.OldClusterDisabled)
		assert.False(t, facts.ConversionDone, "only the new cluster can carry this attempt's marker")
		assert.Equal(t, recoveryReprovision, classifyOutcome(facts),
			"an interrupted transfer must be reprovisioned however the old data directory is decorated")
	})
}

// testAppConfig builds an AppConfig rooted in a temporary directory.
func testAppConfig(t *testing.T) *resources.AppConfig {
	t.Helper()

	pool := &resources.Pool{MountDir: t.TempDir(), PoolDirName: "pool", DataSubDir: "data"}

	return &resources.AppConfig{CloneName: "c1", Branch: "main", Pool: pool}
}

func TestSettledClone(t *testing.T) {
	t.Run("reports the major the data directory actually holds", func(t *testing.T) {
		appConfig := testAppConfig(t)
		require.NoError(t, os.MkdirAll(appConfig.DataDir(), 0700))
		require.NoError(t, os.WriteFile(path.Join(appConfig.DataDir(), "PG_VERSION"), []byte("17"), 0644))

		result := settledClone(appConfig)

		assert.Equal(t, UpgradeUnchanged, result.Outcome)
		assert.Equal(t, "17", result.Version)
	})

	t.Run("an unreadable data directory still settles the clone", func(t *testing.T) {
		result := settledClone(testAppConfig(t))

		assert.Equal(t, UpgradeUnchanged, result.Outcome)
		assert.Empty(t, result.Version)
	})
}

func TestSwapDataDirs(t *testing.T) {
	writeCluster := func(t *testing.T, dir, version string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(dir, 0700))
		require.NoError(t, os.WriteFile(path.Join(dir, "PG_VERSION"), []byte(version), 0644))
	}

	t.Run("promotes the converted cluster and keeps the old one", func(t *testing.T) {
		appConfig := testAppConfig(t)
		writeCluster(t, appConfig.DataDir(), "16")
		writeCluster(t, newDataDir(appConfig), "17")

		require.NoError(t, swapDataDirs(appConfig))

		assert.FileExists(t, path.Join(appConfig.DataDir(), "PG_VERSION"))
		assert.FileExists(t, path.Join(backupDataDir(appConfig), "PG_VERSION"))
		assert.NoDirExists(t, newDataDir(appConfig))
	})

	// The marker travels with the cluster it describes, which is what lets completeUpgrade find it
	// in the data directory afterwards and clear it. Nothing classifies on it once it is there -
	// a promoted data directory is decided by SwapCompleted - but it must not be left behind.
	t.Run("carries the completion marker across with the cluster", func(t *testing.T) {
		appConfig := testAppConfig(t)
		writeCluster(t, appConfig.DataDir(), "16")
		writeCluster(t, newDataDir(appConfig), "17")
		require.NoError(t, os.WriteFile(path.Join(newDataDir(appConfig), upgradeDoneMarkerName), nil, 0644))

		require.NoError(t, swapDataDirs(appConfig))

		assert.FileExists(t, path.Join(appConfig.DataDir(), upgradeDoneMarkerName))
	})

	t.Run("is a no-op once promoted", func(t *testing.T) {
		appConfig := testAppConfig(t)
		writeCluster(t, appConfig.DataDir(), "17")

		require.NoError(t, swapDataDirs(appConfig))

		assert.FileExists(t, path.Join(appConfig.DataDir(), "PG_VERSION"))
	})

	t.Run("finishes a swap interrupted between the two renames", func(t *testing.T) {
		appConfig := testAppConfig(t)
		writeCluster(t, newDataDir(appConfig), "17")
		writeCluster(t, backupDataDir(appConfig), "16")

		require.NoError(t, swapDataDirs(appConfig))

		assert.FileExists(t, path.Join(appConfig.DataDir(), "PG_VERSION"))
		assert.NoDirExists(t, newDataDir(appConfig))
	})

	t.Run("clears a stale backup instead of failing", func(t *testing.T) {
		appConfig := testAppConfig(t)
		writeCluster(t, appConfig.DataDir(), "16")
		writeCluster(t, newDataDir(appConfig), "17")
		writeCluster(t, backupDataDir(appConfig), "15")

		require.NoError(t, swapDataDirs(appConfig))

		backupVersion, err := os.ReadFile(path.Join(backupDataDir(appConfig), "PG_VERSION"))
		require.NoError(t, err)
		assert.Equal(t, "16", string(backupVersion), "the backup must be the cluster we just replaced")
	})
}

func TestDescribeExitCode(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		expected string
	}{
		{name: "pre-transfer names the safe case", exitCode: exitPreTransfer, expected: "before converting anything"},
		{name: "post-transfer names the unsafe case", exitCode: exitPostTransfer, expected: "after conversion had begun"},
		{name: "an unrecognised code is still reported", exitCode: 137, expected: "exit code 137"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, describeExitCode(tt.exitCode), tt.expected)
		})
	}
}

func TestUpgradeContainerEnv(t *testing.T) {
	state := UpgradeState{OldVersion: "16", NewVersion: "17"}

	t.Run("libc locale omits provider and ICU entries", func(t *testing.T) {
		req := UpgradeRequest{Encoding: "UTF8", LCCollate: "en_US.utf8", LCCtype: "en_US.utf8", DataChecksums: "on"}

		env := upgradeContainerEnv("/pool/clones/main/c1/r0", "data", state, req)

		assert.Contains(t, env, "CLONE_DIR=/pool/clones/main/c1/r0")
		assert.Contains(t, env, "DATA_SUBDIR=data")
		assert.Contains(t, env, "OLD_VERSION=16")
		assert.Contains(t, env, "NEW_VERSION=17")
		assert.Contains(t, env, "ENCODING=UTF8")
		assert.Contains(t, env, "LC_COLLATE_VALUE=en_US.utf8")
		assert.Contains(t, env, "LC_CTYPE_VALUE=en_US.utf8")
		assert.Contains(t, env, "DATA_CHECKSUMS=on")

		for _, pair := range env {
			assert.NotContains(t, pair, "LOCALE_PROVIDER=")
			assert.NotContains(t, pair, "ICU_LOCALE=")
			assert.NotContains(t, pair, "OLD_SERVER_OPTIONS=")
		}
	})

	t.Run("icu provider carries the locale", func(t *testing.T) {
		req := UpgradeRequest{Encoding: "UTF8", LCCollate: "C", LCCtype: "C", DataChecksums: "off",
			LocaleProvider: "icu", ICULocale: "en-US"}

		env := upgradeContainerEnv("/pool/c1", "data", state, req)

		assert.Contains(t, env, "LOCALE_PROVIDER=icu")
		assert.Contains(t, env, "ICU_LOCALE=en-US")
	})

	t.Run("old server options are passed through", func(t *testing.T) {
		req := UpgradeRequest{Encoding: "UTF8", LCCollate: "C", LCCtype: "C",
			OldServerOptions: "-c shared_preload_libraries=''"}

		env := upgradeContainerEnv("/pool/c1", "data", state, req)

		assert.Contains(t, env, "OLD_SERVER_OPTIONS=-c shared_preload_libraries=''")
	})

	t.Run("honours a non-default data subdir", func(t *testing.T) {
		env := upgradeContainerEnv("/pool/c1", "pgdata", state, UpgradeRequest{})

		assert.Contains(t, env, "DATA_SUBDIR=pgdata")
	})

	t.Run("never uses the real POSIX locale variable names", func(t *testing.T) {
		req := UpgradeRequest{LCCollate: "en_US.utf8", LCCtype: "en_US.utf8"}

		for _, pair := range upgradeContainerEnv("/pool/c1", "data", state, req) {
			assert.False(t, strings.HasPrefix(pair, "LC_COLLATE="), "would change the container process locale")
			assert.False(t, strings.HasPrefix(pair, "LC_CTYPE="), "would change the container process locale")
		}
	})
}

func TestDataDirOwner(t *testing.T) {
	tests := []struct {
		name     string
		out      string
		err      error
		expected string
		wantErr  string
	}{
		{name: "regular owner", out: "999:999\n", expected: "999:999"},
		{name: "engine user owner", out: "1000:1000", expected: "1000:1000"},
		{name: "root is rejected", out: "0:0", wantErr: "cannot run as root"},
		{name: "empty output is rejected", out: "", wantErr: "cannot run as root"},
		{name: "stat failure", err: errors.New("no such file"), wantErr: "failed to detect the owner"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, err := dataDirOwner(&stubRunner{out: tt.out, err: tt.err}, "/pool/clones/main/c1/r0/data")

			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, owner)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestDataDirOwnerQuotesThePath(t *testing.T) {
	runner := &stubRunner{out: "999:999"}

	_, err := dataDirOwner(runner, "/pool/clones/main/it's odd/r0/data")

	require.NoError(t, err)
	assert.Equal(t, `stat -c '%u:%g' '/pool/clones/main/it'\''s odd/r0/data'`, runner.lastCmd)
}

func TestUpgradeStateRoundTrip(t *testing.T) {
	cloneDir := t.TempDir()
	require.NoError(t, os.MkdirAll(upgradeDirPath(cloneDir), 0755))
	state := UpgradeState{
		OldVersion:  "16",
		NewVersion:  "17",
		OldImage:    "postgresai/extended-postgres:16-0.8.0",
		TargetImage: "postgresai/extended-postgres:17-0.8.0",
		SnapshotID:  "pool@snapshot_20260813",
	}

	require.NoError(t, writeUpgradeState(cloneDir, state))

	restored, err := readUpgradeState(cloneDir)
	require.NoError(t, err)
	assert.Equal(t, state, restored)
}

func TestReadUpgradeStateMissingFile(t *testing.T) {
	_, err := readUpgradeState(t.TempDir())

	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestUpgradeStageFile(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		cloneDir := t.TempDir()
		require.NoError(t, os.MkdirAll(upgradeDirPath(cloneDir), 0755))
		require.NoError(t, writeUpgradeStage(cloneDir, stageTransfer))
		assert.Equal(t, stageTransfer, readUpgradeStage(cloneDir))
	})

	t.Run("missing file reads as unknown", func(t *testing.T) {
		assert.Equal(t, stageUnknown, readUpgradeStage(t.TempDir()))
	})

	t.Run("empty file reads as unknown", func(t *testing.T) {
		cloneDir := t.TempDir()
		require.NoError(t, os.MkdirAll(upgradeDirPath(cloneDir), 0755))
		require.NoError(t, os.WriteFile(path.Join(upgradeDirPath(cloneDir), upgradeStageFileName), []byte("  \n"), 0600))
		assert.Equal(t, stageUnknown, readUpgradeStage(cloneDir))
	})

	t.Run("trailing newline is tolerated", func(t *testing.T) {
		cloneDir := t.TempDir()
		require.NoError(t, os.MkdirAll(upgradeDirPath(cloneDir), 0755))
		require.NoError(t, os.WriteFile(path.Join(upgradeDirPath(cloneDir), upgradeStageFileName), []byte("done\n"), 0600))
		assert.Equal(t, stageDone, readUpgradeStage(cloneDir))
	})
}

func TestHasPendingUpgrade(t *testing.T) {
	t.Run("no upgrade directory at all", func(t *testing.T) {
		assert.False(t, hasPendingUpgrade(t.TempDir()))
	})

	t.Run("state written before the clone is stopped", func(t *testing.T) {
		cloneDir := t.TempDir()
		require.NoError(t, os.MkdirAll(upgradeDirPath(cloneDir), 0755))
		require.NoError(t, writeUpgradeState(cloneDir, UpgradeState{OldVersion: "16", NewVersion: "17"}))

		assert.True(t, hasPendingUpgrade(cloneDir))
	})

	t.Run("settled upgrade leaves nothing to recover", func(t *testing.T) {
		cloneDir := t.TempDir()
		require.NoError(t, os.MkdirAll(upgradeDirPath(cloneDir), 0755))
		require.NoError(t, writeUpgradeState(cloneDir, UpgradeState{OldVersion: "16", NewVersion: "17"}))

		clearUpgradeArtifacts(cloneDir)

		assert.False(t, hasPendingUpgrade(cloneDir))
	})
}

func TestMarkRecoverable(t *testing.T) {
	appConfig := &resources.AppConfig{
		CloneName: "c1",
		Branch:    "main",
		Pool:      &resources.Pool{MountDir: t.TempDir(), PoolDirName: "pool", DataSubDir: "data"},
	}

	require.NoError(t, os.MkdirAll(upgradeDirPath(appConfig.CloneDir()), 0755))

	t.Run("success is never tagged", func(t *testing.T) {
		require.NoError(t, writeUpgradeState(appConfig.CloneDir(), UpgradeState{OldVersion: "16"}))

		result, err := markRecoverable(appConfig, UpgradeResult{Outcome: UpgradeSucceeded}, nil)

		require.NoError(t, err)
		assert.False(t, result.Recoverable)
	})

	t.Run("failure with the state still on disk is recoverable", func(t *testing.T) {
		require.NoError(t, writeUpgradeState(appConfig.CloneDir(), UpgradeState{OldVersion: "16"}))

		result, err := markRecoverable(appConfig, UpgradeResult{}, errors.New("failed to start the upgraded clone"))

		require.Error(t, err)
		assert.True(t, result.Recoverable, "the next engine start finishes an upgrade whose state survived")
	})

	t.Run("failure after the upgrade settled is terminal", func(t *testing.T) {
		clearUpgradeArtifacts(appConfig.CloneDir())

		result, err := markRecoverable(appConfig, UpgradeResult{}, errors.New("failed to find filesystem manager"))

		require.Error(t, err)
		assert.False(t, result.Recoverable)
	})
}

func TestAbortPreparedUpgrade(t *testing.T) {
	appConfig := &resources.AppConfig{
		CloneName: "c1",
		Branch:    "main",
		Pool:      &resources.Pool{MountDir: t.TempDir(), PoolDirName: "pool", DataSubDir: "data"},
	}
	logDir := path.Join(upgradeDirPath(appConfig.CloneDir()), upgradeLogsDirName)

	require.NoError(t, os.MkdirAll(logDir, 0755))
	require.NoError(t, os.MkdirAll(newDataDir(appConfig), 0700))
	require.NoError(t, os.WriteFile(path.Join(logDir, upgradeLogFileName), []byte("log line"), 0600))
	require.NoError(t, writeUpgradeState(appConfig.CloneDir(), UpgradeState{OldVersion: "16", NewVersion: "17"}))
	require.NoError(t, writeUpgradeStage(appConfig.CloneDir(), stagePrepare))

	result := abortPreparedUpgrade(appConfig, UpgradeState{OldVersion: "16", OldImage: "img:16"}, "the clone could not be stopped")

	assert.Equal(t, UpgradeRolledBack, result.Outcome)
	assert.Equal(t, "the clone could not be stopped", result.Cause)
	assert.False(t, hasPendingUpgrade(appConfig.CloneDir()),
		"a clone left running must carry no state that makes the next start recover it")
	assert.NoDirExists(t, newDataDir(appConfig), "a snapshot taken now would otherwise carry an empty cluster")
	assert.FileExists(t, path.Join(logDir, upgradeLogFileName), "the log is the user's diagnostic and must survive")
}

func TestUpgradeTimeout(t *testing.T) {
	assert.Equal(t, defaultUpgradeTimeout, (&Provisioner{config: &Config{}}).upgradeTimeout())
	assert.Equal(t, time.Hour, (&Provisioner{config: &Config{PgUpgradeTimeout: time.Hour}}).upgradeTimeout())
}

func TestUpgradePullTimeout(t *testing.T) {
	assert.Equal(t, defaultUpgradePullTimeout, (&Provisioner{config: &Config{}}).upgradePullTimeout())
	assert.Equal(t, time.Minute, (&Provisioner{config: &Config{PgUpgradePullTimeout: time.Minute}}).upgradePullTimeout())

	// the two budgets are independent: raising the conversion budget must not widen the pull.
	both := &Provisioner{config: &Config{PgUpgradeTimeout: 5 * time.Hour}}
	assert.Equal(t, defaultUpgradePullTimeout, both.upgradePullTimeout())
}

func TestClearUpgradeArtifactsKeepsLogs(t *testing.T) {
	cloneDir := t.TempDir()
	logDir := path.Join(upgradeDirPath(cloneDir), upgradeLogsDirName)

	require.NoError(t, os.MkdirAll(logDir, 0755))
	require.NoError(t, os.WriteFile(path.Join(logDir, upgradeLogFileName), []byte("log line"), 0600))
	require.NoError(t, os.MkdirAll(upgradeDirPath(cloneDir), 0755))
	require.NoError(t, writeUpgradeState(cloneDir, UpgradeState{OldVersion: "16"}))
	require.NoError(t, writeUpgradeStage(cloneDir, stageDone))

	clearUpgradeArtifacts(cloneDir)

	assert.NoFileExists(t, path.Join(upgradeDirPath(cloneDir), upgradeStateFileName))
	assert.NoFileExists(t, path.Join(upgradeDirPath(cloneDir), upgradeStageFileName))
	assert.FileExists(t, path.Join(logDir, upgradeLogFileName), "the log is the user's diagnostic and must survive")
}

func TestClearUpgradeArtifactsOnMissingFiles(t *testing.T) {
	assert.NotPanics(t, func() { clearUpgradeArtifacts(t.TempDir()) })
}

func TestReadLogTail(t *testing.T) {
	t.Run("missing log yields an empty tail", func(t *testing.T) {
		assert.Empty(t, readLogTail(t.TempDir()))
	})

	t.Run("keeps only the last lines", func(t *testing.T) {
		cloneDir := t.TempDir()
		logDir := path.Join(upgradeDirPath(cloneDir), upgradeLogsDirName)
		require.NoError(t, os.MkdirAll(logDir, 0755))

		content := ""
		for i := 0; i < logTailLines+15; i++ {
			content += "line" + strconv.Itoa(i) + "\n"
		}

		require.NoError(t, os.WriteFile(path.Join(logDir, upgradeLogFileName), []byte(content), 0600))

		tail := readLogTail(cloneDir)

		assert.Equal(t, logTailLines, len(strings.Split(tail, "\n")))
		assert.Contains(t, tail, "line34")
		assert.NotContains(t, tail, "line0\n")
	})
}

func TestFormatPGVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  float64
		expected string
	}{
		{name: "modern major", version: 17, expected: "17"},
		{name: "legacy dotted", version: 9.6, expected: "9.6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatPGVersion(tt.version))
		})
	}
}

func TestUpgradeContainerName(t *testing.T) {
	assert.Equal(t, "dblab_upgrade_clone123", upgradeContainerName("clone123"))
}

func TestRunnersQuote(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{name: "plain value", value: "UTF8", expected: `'UTF8'`},
		{name: "value with spaces", value: "-c shared_preload_libraries=", expected: `'-c shared_preload_libraries='`},
		{name: "embedded single quote", value: "it's", expected: `'it'\''s'`},
		{name: "shell metacharacters stay data", value: "a; rm -rf /", expected: `'a; rm -rf /'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, runners.Quote(tt.value))
		})
	}
}

// recordingRunner captures the commands prepareUpgradeWorkspace issues.
type recordingRunner struct {
	commands []string
}

func (r *recordingRunner) Run(command string, _ ...bool) (string, error) {
	r.commands = append(r.commands, command)

	return "", nil
}

// TestPrepareUpgradeWorkspaceHandsOverOnlyWhatTheContainerWrites pins the ownership boundary.
//
// It cannot be covered by the integration tests: those run the container as the same uid that owns
// the clone directory, so the chown is a no-op there, while in a real deployment the clone
// directory belongs to the engine user and the container runs as the data directory's owner. The
// upgrade directory must stay with the engine because it holds state.json, which names the images
// RecoverUpgrade starts the clone on - a container able to create files there could replace it.
func TestPrepareUpgradeWorkspaceHandsOverOnlyWhatTheContainerWrites(t *testing.T) {
	appConfig := testAppConfig(t)
	require.NoError(t, os.MkdirAll(appConfig.DataDir(), 0700))

	runner := &recordingRunner{}
	p := &Provisioner{runner: runner}

	require.NoError(t, p.prepareUpgradeWorkspace(appConfig, "999:999", UpgradeState{OldVersion: "16", NewVersion: "17"}))

	upgradeDir := upgradeDirPath(appConfig.CloneDir())
	expected := []string{
		fmt.Sprintf("chown -R 999:999 %s", runners.Quote(path.Join(upgradeDir, upgradeLogsDirName))),
		fmt.Sprintf("chown -R 999:999 %s", runners.Quote(path.Join(upgradeDir, upgradeStageFileName))),
		fmt.Sprintf("chown -R 999:999 %s", runners.Quote(newDataDir(appConfig))),
	}

	// The ordered comparison is the whole assertion: it fails both if the upgrade directory is
	// handed over again and if any of the three narrower targets stops being.
	assert.Equal(t, expected, runner.commands,
		"only the log directory, the stage file and the new cluster may change owner; "+
			"the upgrade directory holds state.json and must stay with the engine")
}

// TestUpgradeScriptDeclaresTheSameMarkerName pins the one string the engine and the upgrade script
// must agree on. classifyOutcome promotes a recovered upgrade only when it finds this file, so a
// rename on either side turns an interrupted-then-recovered upgrade into a reprovision - the clone
// is rebuilt from its snapshot and every write made to it since is silently discarded.
//
// The integration tests assert the same coupling against the real script, but they skip wherever
// the docker daemon does not share this filesystem, which includes CI today. This check needs no
// docker and runs in every `go test ./internal/...`.
func TestUpgradeScriptDeclaresTheSameMarkerName(t *testing.T) {
	script, err := os.ReadFile(path.Join("..", "..", "scripts", "pg_upgrade_clone.sh"))
	require.NoError(t, err)

	assert.Contains(t, string(script), `DONE_MARKER_NAME="`+upgradeDoneMarkerName+`"`,
		"scripts/pg_upgrade_clone.sh must declare the marker name the engine looks for")
}

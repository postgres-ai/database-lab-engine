/*
2026 © Postgres.ai
*/

package provision

import (
	"bytes"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

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

	t.Run("does not follow a link left in place of the log", func(t *testing.T) {
		cloneDir := t.TempDir()
		logDir := path.Join(upgradeDirPath(cloneDir), upgradeLogsDirName)
		require.NoError(t, os.MkdirAll(logDir, 0755))

		secret := path.Join(t.TempDir(), "secret")
		require.NoError(t, os.WriteFile(secret, []byte("not for the status\n"), 0600))
		require.NoError(t, os.Symlink(secret, path.Join(logDir, upgradeLogFileName)))

		assert.Empty(t, readLogTail(cloneDir))
	})

	t.Run("reads only the end of a log beyond the limit", func(t *testing.T) {
		cloneDir := t.TempDir()
		logDir := path.Join(upgradeDirPath(cloneDir), upgradeLogsDirName)
		require.NoError(t, os.MkdirAll(logDir, 0755))

		content := append(bytes.Repeat([]byte("padding line\n"), upgradeLogReadLimit/len("padding line\n")+1), []byte("the end\n")...)
		require.Greater(t, len(content), upgradeLogReadLimit)
		require.NoError(t, os.WriteFile(path.Join(logDir, upgradeLogFileName), content, 0600))

		tail := readLogTail(cloneDir)

		assert.Equal(t, logTailLines, len(strings.Split(tail, "\n")))
		assert.True(t, strings.HasSuffix(tail, "the end"), tail)
	})

	t.Run("a fifo in place of the log yields an empty tail without blocking", func(t *testing.T) {
		cloneDir := t.TempDir()
		logDir := path.Join(upgradeDirPath(cloneDir), upgradeLogsDirName)
		require.NoError(t, os.MkdirAll(logDir, 0755))
		require.NoError(t, unix.Mkfifo(path.Join(logDir, upgradeLogFileName), 0600))

		assert.Empty(t, readPromptly(t, func() string { return readLogTail(cloneDir) }))
	})
}

func TestPreserveUpgradeOutput(t *testing.T) {
	t.Run("nothing to preserve", func(t *testing.T) {
		appConfig := testAppConfig(t)

		assert.Empty(t, preserveUpgradeOutput(appConfig))
		assert.NoDirExists(t, upgradeOutputPath(appConfig.CloneDir()))
	})

	t.Run("moves every run and keeps the runs of earlier attempts", func(t *testing.T) {
		appConfig := testAppConfig(t)
		prepareUpgradeLogs(t, appConfig.CloneDir())
		source := path.Join(newDataDir(appConfig), upgradeOutputDirName)
		target := upgradeOutputPath(appConfig.CloneDir())

		writeServerLog(t, source, "20260101T000000.000", "FATAL:  first attempt\n")
		assert.Equal(t, []string{"20260101T000000.000"}, preserveUpgradeOutput(appConfig))

		writeServerLog(t, source, "20260102T000000.000", "FATAL:  second attempt\n")
		writeServerLog(t, source, "20260103T000000.000", "FATAL:  second attempt, retried\n")
		assert.Equal(t, []string{"20260102T000000.000", "20260103T000000.000"}, preserveUpgradeOutput(appConfig))

		assert.FileExists(t, path.Join(target, "20260101T000000.000", upgradeServerLogDirName, upgradeServerLogName))
		assert.FileExists(t, path.Join(target, "20260102T000000.000", upgradeServerLogDirName, upgradeServerLogName))
		assert.FileExists(t, path.Join(target, "20260103T000000.000", upgradeServerLogDirName, upgradeServerLogName))

		left, err := os.ReadDir(source)
		require.NoError(t, err)
		assert.Empty(t, left, "the new cluster is discarded right after, nothing must be left to lose")
	})

	t.Run("an output directory without runs preserves nothing", func(t *testing.T) {
		appConfig := testAppConfig(t)
		source := path.Join(newDataDir(appConfig), upgradeOutputDirName)
		require.NoError(t, os.MkdirAll(source, 0755))
		require.NoError(t, os.WriteFile(path.Join(source, "stray.txt"), []byte("x"), 0600))

		assert.Empty(t, preserveUpgradeOutput(appConfig))
		assert.FileExists(t, path.Join(source, "stray.txt"))
	})

	t.Run("prunes the oldest runs beyond the limit", func(t *testing.T) {
		appConfig := testAppConfig(t)
		prepareUpgradeLogs(t, appConfig.CloneDir())
		source := path.Join(newDataDir(appConfig), upgradeOutputDirName)
		target := upgradeOutputPath(appConfig.CloneDir())

		for i := 0; i < preservedUpgradeRuns+2; i++ {
			writeServerLog(t, source, "2026010"+strconv.Itoa(i)+"T000000.000", "FATAL:  attempt\n")
		}

		moved := preserveUpgradeOutput(appConfig)
		assert.Len(t, moved, preservedUpgradeRuns+2)

		kept, err := os.ReadDir(target)
		require.NoError(t, err)
		assert.Len(t, kept, preservedUpgradeRuns)
		assert.NoDirExists(t, path.Join(target, "20260100T000000.000"))
		assert.NoDirExists(t, path.Join(target, "20260101T000000.000"))
		assert.DirExists(t, path.Join(target, "20260102T000000.000"))
	})

	t.Run("a run that cannot be moved is left where it is and not reported", func(t *testing.T) {
		appConfig := testAppConfig(t)
		prepareUpgradeLogs(t, appConfig.CloneDir())
		source := path.Join(newDataDir(appConfig), upgradeOutputDirName)
		target := upgradeOutputPath(appConfig.CloneDir())

		writeServerLog(t, source, "20260101T000000.000", "FATAL:  first\n")
		writeServerLog(t, source, "20260102T000000.000", "FATAL:  second\n")
		writeServerLog(t, source, "20260103T000000.000", "FATAL:  third\n")
		writeServerLog(t, target, "20260102T000000.000", "FATAL:  already there, not empty\n")

		assert.Equal(t, []string{"20260101T000000.000", "20260103T000000.000"}, preserveUpgradeOutput(appConfig))

		assert.FileExists(t, path.Join(source, "20260102T000000.000", upgradeServerLogDirName, upgradeServerLogName))
		assert.NoDirExists(t, path.Join(source, "20260101T000000.000"))
		assert.NoDirExists(t, path.Join(source, "20260103T000000.000"))

		kept, err := os.ReadFile(path.Join(target, "20260102T000000.000", upgradeServerLogDirName, upgradeServerLogName))
		require.NoError(t, err)
		assert.Equal(t, "FATAL:  already there, not empty\n", string(kept))
	})

	t.Run("pruning removes nested directories and does not follow a link inside a run", func(t *testing.T) {
		appConfig := testAppConfig(t)
		prepareUpgradeLogs(t, appConfig.CloneDir())
		source := path.Join(newDataDir(appConfig), upgradeOutputDirName)
		target := upgradeOutputPath(appConfig.CloneDir())

		outside := t.TempDir()
		require.NoError(t, os.WriteFile(path.Join(outside, "marker"), []byte("keep"), 0600))

		oldest := path.Join(target, "20260100T000000.000")
		require.NoError(t, os.MkdirAll(path.Join(oldest, "dump", "nested", "deeper"), 0755))
		require.NoError(t, os.WriteFile(path.Join(oldest, "dump", "nested", "deeper", "file"), []byte("x"), 0600))
		require.NoError(t, os.Symlink(outside, path.Join(oldest, "evil")))
		require.NoError(t, os.Symlink("/", path.Join(oldest, "dump", "root")))

		for i := 1; i <= preservedUpgradeRuns; i++ {
			writeServerLog(t, source, "2026010"+strconv.Itoa(i)+"T000000.000", "FATAL:  attempt\n")
		}

		assert.Len(t, preserveUpgradeOutput(appConfig), preservedUpgradeRuns)

		assert.NoDirExists(t, oldest)
		assert.FileExists(t, path.Join(outside, "marker"), "a link inside a pruned run must not be followed")
		assert.DirExists(t, path.Join(target, "20260101T000000.000"))
	})

	t.Run("does not follow a link planted in place of the preserved directory", func(t *testing.T) {
		appConfig := testAppConfig(t)
		source := path.Join(newDataDir(appConfig), upgradeOutputDirName)
		upgradeDir := upgradeDirPath(appConfig.CloneDir())
		logsDir := path.Join(upgradeDir, upgradeLogsDirName)
		require.NoError(t, os.MkdirAll(logsDir, 0755))
		require.NoError(t, os.WriteFile(path.Join(upgradeDir, upgradeStateFileName), []byte("{}"), 0600))
		require.NoError(t, os.Symlink("..", path.Join(logsDir, upgradeOutputDirName)))

		writeServerLog(t, source, upgradeStateFileName, "FATAL:  planted\n")

		assert.Empty(t, preserveUpgradeOutput(appConfig))

		state, err := os.ReadFile(path.Join(upgradeDir, upgradeStateFileName))
		require.NoError(t, err)
		assert.Equal(t, "{}", string(state), "the engine's state file must survive a link in the log directory")
		assert.DirExists(t, path.Join(source, upgradeStateFileName), "the run stays where it was")
	})

	t.Run("does not follow a link planted in place of the output directory", func(t *testing.T) {
		appConfig := testAppConfig(t)
		newDir := newDataDir(appConfig)
		require.NoError(t, os.MkdirAll(newDir, 0700))
		require.NoError(t, os.MkdirAll(appConfig.DataDir(), 0700))
		require.NoError(t, os.Symlink("..", path.Join(newDir, upgradeOutputDirName)))

		assert.Empty(t, preserveUpgradeOutput(appConfig))
		assert.DirExists(t, appConfig.DataDir(), "the data directory must not be moved away")
		assert.NoDirExists(t, upgradeOutputPath(appConfig.CloneDir()))
	})

	t.Run("leaves a link among the runs where it is", func(t *testing.T) {
		appConfig := testAppConfig(t)
		prepareUpgradeLogs(t, appConfig.CloneDir())
		source := path.Join(newDataDir(appConfig), upgradeOutputDirName)
		writeServerLog(t, source, "20260101T000000.000", "FATAL:  real run\n")
		require.NoError(t, os.Symlink("/", path.Join(source, "20260102T000000.000")))

		assert.Equal(t, []string{"20260101T000000.000"}, preserveUpgradeOutput(appConfig))
		assert.NoFileExists(t, path.Join(upgradeOutputPath(appConfig.CloneDir()), "20260102T000000.000"))
	})
}

func TestReadServerLogError(t *testing.T) {
	const run = "20260102T000000.000"

	t.Run("no preserved run yields nothing", func(t *testing.T) {
		cloneDir := t.TempDir()
		writeServerLog(t, upgradeOutputPath(cloneDir), "20260101T000000.000", "FATAL:  older attempt\n")

		assert.Empty(t, readServerLogError(cloneDir, nil))
	})

	t.Run("a run without a server log yields nothing", func(t *testing.T) {
		cloneDir := t.TempDir()
		require.NoError(t, os.MkdirAll(path.Join(upgradeOutputPath(cloneDir), run, upgradeServerLogDirName), 0755))

		assert.Empty(t, readServerLogError(cloneDir, []string{run}))
	})

	t.Run("a server log without a fatal line yields nothing", func(t *testing.T) {
		cloneDir := t.TempDir()
		writeServerLog(t, upgradeOutputPath(cloneDir), run, "LOG:  starting PostgreSQL 16\nLOG:  a message that mentions FATAL: in passing\n")

		assert.Empty(t, readServerLogError(cloneDir, []string{run}))
	})

	t.Run("takes the first fatal line of the newest preserved run with its detail", func(t *testing.T) {
		cloneDir := t.TempDir()
		writeServerLog(t, upgradeOutputPath(cloneDir), "20260101T000000.000", "FATAL:  older attempt\n")
		writeServerLog(t, upgradeOutputPath(cloneDir), run, strings.Join([]string{
			"2026-01-02 00:00:00.000 UTC [12] LOG:  starting PostgreSQL 16.4",
			"2026-01-02 00:00:00.001 UTC [12] FATAL:  could not load server certificate file \"server.crt\": No such file",
			"2026-01-02 00:00:00.001 UTC [12] DETAIL:  the certificate volume is not mounted",
			"2026-01-02 00:00:00.001 UTC [12] HINT:  mount it",
			"2026-01-02 00:00:00.002 UTC [12] LOG:  database system is shut down",
			"2026-01-02 00:00:00.003 UTC [13] FATAL:  terminating connection due to administrator command",
			"",
		}, "\n"))

		assert.Equal(t, "FATAL:  could not load server certificate file \"server.crt\": No such file; "+
			"DETAIL:  the certificate volume is not mounted; HINT:  mount it",
			readServerLogError(cloneDir, []string{"20260101T000000.000", run}))
	})

	t.Run("keeps at most two continuation lines and strips control characters", func(t *testing.T) {
		cloneDir := t.TempDir()
		writeServerLog(t, upgradeOutputPath(cloneDir), run, strings.Join([]string{
			"FATAL:  bad \x1b[31mred\x1b[0m thing\r",
			"DETAIL:  one",
			"HINT:  two",
			"HINT:  three",
			"",
		}, "\n"))

		assert.Equal(t, "FATAL:  bad [31mred[0m thing; DETAIL:  one; HINT:  two", readServerLogError(cloneDir, []string{run}))
	})

	t.Run("truncates a long error", func(t *testing.T) {
		cloneDir := t.TempDir()
		writeServerLog(t, upgradeOutputPath(cloneDir), run, "FATAL:  "+strings.Repeat("é", serverErrorLengthLimit)+"\n")

		got := readServerLogError(cloneDir, []string{run})

		assert.LessOrEqual(t, len(got), serverErrorLengthLimit)
		assert.True(t, strings.HasSuffix(got, serverErrorEllipsis))
		assert.True(t, strings.HasPrefix(got, "FATAL:  éé"))
	})

	t.Run("a fatal line beyond the read limit is not found", func(t *testing.T) {
		cloneDir := t.TempDir()
		content := append(bytes.Repeat([]byte("LOG:  noise\n"), upgradeLogReadLimit/len("LOG:  noise\n")+1), []byte("FATAL:  too late\n")...)
		require.Greater(t, len(content), upgradeLogReadLimit)
		writeServerLog(t, upgradeOutputPath(cloneDir), run, string(content))

		assert.Empty(t, readServerLogError(cloneDir, []string{run}))
	})

	t.Run("a fifo in place of the server log yields nothing without blocking", func(t *testing.T) {
		cloneDir := t.TempDir()
		logDir := path.Join(upgradeOutputPath(cloneDir), run, upgradeServerLogDirName)
		require.NoError(t, os.MkdirAll(logDir, 0755))
		require.NoError(t, unix.Mkfifo(path.Join(logDir, upgradeServerLogName), 0600))

		assert.Empty(t, readPromptly(t, func() string { return readServerLogError(cloneDir, []string{run}) }))
	})

	t.Run("a panic counts as well", func(t *testing.T) {
		cloneDir := t.TempDir()
		writeServerLog(t, upgradeOutputPath(cloneDir), run, "PANIC:  could not locate a valid checkpoint record\n")

		assert.Equal(t, "PANIC:  could not locate a valid checkpoint record", readServerLogError(cloneDir, []string{run}))
	})

	t.Run("does not follow a link left in place of the server log", func(t *testing.T) {
		cloneDir := t.TempDir()
		logDir := path.Join(upgradeOutputPath(cloneDir), run, upgradeServerLogDirName)
		require.NoError(t, os.MkdirAll(logDir, 0755))

		secret := path.Join(t.TempDir(), "secret")
		require.NoError(t, os.WriteFile(secret, []byte("FATAL:  not for the status\n"), 0600))
		require.NoError(t, os.Symlink(secret, path.Join(logDir, upgradeServerLogName)))

		assert.Empty(t, readServerLogError(cloneDir, []string{run}))
	})
}

// prepareUpgradeLogs creates the upgrade log directory the way prepareUpgradeWorkspace does before
// the container runs: preserving never creates it, so a test that expects a move needs it.
func prepareUpgradeLogs(t *testing.T, cloneDir string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(path.Join(upgradeDirPath(cloneDir), upgradeLogsDirName), 0755))
}

// readPromptly runs a log reader and fails the test when it does not return within a few seconds,
// which is what a blocking open on a fifo would do.
func readPromptly(t *testing.T, read func() string) string {
	t.Helper()

	result := make(chan string, 1)

	go func() { result <- read() }()

	select {
	case got := <-result:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("reading the log blocked")

		return ""
	}
}

func writeServerLog(t *testing.T, outputDir, run, content string) {
	t.Helper()

	logDir := path.Join(outputDir, run, upgradeServerLogDirName)
	require.NoError(t, os.MkdirAll(logDir, 0755))
	require.NoError(t, os.WriteFile(path.Join(logDir, upgradeServerLogName), []byte(content), 0600))
}

func TestDescribeFailure(t *testing.T) {
	assert.Equal(t, "exit code 10", describeFailure("exit code 10", ""))
	assert.Equal(t, "exit code 10; the postmaster started by pg_upgrade reported: FATAL:  boom", describeFailure("exit code 10", "FATAL:  boom"))
}

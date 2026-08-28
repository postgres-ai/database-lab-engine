//go:build integration
// +build integration

/*
2026 © Postgres.ai
*/

package provision

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/docker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
)

// These tests run the real scripts/pg_upgrade_clone.sh in the real upgrade image against a real
// cluster. What they pin down is the premise the recovery logic rests on and that no unit test can
// reach: the script initdb's the new cluster before pg_upgrade is asked to check the pair, so a
// target-major PG_VERSION in <data>_new appears even for a failure that converted nothing. Only the
// old cluster's pg_control dates the conversion. Reading the new directory as proof of success
// promotes an empty initdb over the clone's data and then deletes the data.
const (
	// The image carries the four majors preceding its own, so any of them serves as a source.
	upgradeSourceMajor = "16"
	upgradeTargetMajor = "17"

	// A tiny cluster converts in seconds; the budget only has to outlast an image pull.
	upgradeRunTimeout = 10 * time.Minute

	upgradeProbeQuery = "select answer from upgrade_probe"
)

// pgUpgradeImage resolves the image under test. The feature branch pipeline publishes it to the
// project registry, so a run against anything but the released Docker Hub image points
// DBLAB_PG_UPGRADE_IMAGE at that tag.
// upgradeTestsRequired reports whether a skip in these tests must be treated as a failure. CI sets
// it for the job that publishes the upgrade image and is therefore expected to exercise the real
// conversion; without it a topology these tests cannot run in - a daemon that does not share this
// filesystem, an unreachable image - would silently reduce the suite to nothing while the job
// still reported success.
func upgradeTestsRequired() bool {
	return os.Getenv("DBLAB_UPGRADE_TESTS_REQUIRED") == "true"
}

// skipUnlessRequired skips the test, or fails it when this environment is one that must run it.
func skipUnlessRequired(t *testing.T, format string, args ...any) {
	t.Helper()

	if upgradeTestsRequired() {
		t.Fatalf("DBLAB_UPGRADE_TESTS_REQUIRED is set, so this test must run: "+format, args...)
	}

	t.Skipf(format, args...)
}

func pgUpgradeImage() string {
	if image := os.Getenv("DBLAB_PG_UPGRADE_IMAGE"); image != "" {
		return image
	}

	return "postgresai/pg-upgrade:" + upgradeTargetMajor
}

// ensureUpgradeImage makes the image available to the daemon. docker.PrepareImage pulls
// anonymously, which covers the released Docker Hub image but not the per-branch build in the
// project registry, so the CI registry credentials are used whenever the environment carries them.
func ensureUpgradeImage(ctx context.Context, dockerClient *client.Client, image string) error {
	exists, err := docker.ImageExists(ctx, dockerClient, image)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	auth, err := registryAuth()
	if err != nil {
		return err
	}

	if auth == "" {
		return docker.PullImage(ctx, dockerClient, image)
	}

	response, err := dockerClient.ImagePull(ctx, image, client.ImagePullOptions{RegistryAuth: auth})
	if err != nil {
		return fmt.Errorf("failed to pull %s: %w", image, err)
	}

	defer response.Close()

	// The daemon only performs the pull while its progress stream is consumed.
	return response.Wait(ctx)
}

// registryAuth encodes the CI registry credentials the way the Docker API expects them. Missing
// credentials are not an error: a public image needs none.
func registryAuth() (string, error) {
	user, password := os.Getenv("CI_REGISTRY_USER"), os.Getenv("CI_REGISTRY_PASSWORD")
	if user == "" || password == "" {
		return "", nil
	}

	encoded, err := json.Marshal(registry.AuthConfig{
		Username:      user,
		Password:      password,
		ServerAddress: os.Getenv("CI_REGISTRY"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode the registry credentials: %w", err)
	}

	return base64.URLEncoding.EncodeToString(encoded), nil
}

// upgradeFixture is a clone directory laid out the way the engine lays one out, holding a source
// cluster with one row to account for.
type upgradeFixture struct {
	appConfig *resources.AppConfig
	state     UpgradeState
	image     string
	docker    *client.Client
	// hostDir holds the passwd file described in prepareUser. It is deliberately outside the
	// clone directory, which stays exactly as the engine would have left it.
	hostDir     string
	passwdReady bool
}

// newUpgradeFixture builds the source cluster and seeds it with a row to account for. Extra
// statements are run against it before the shutdown, which is how a test arranges a cluster that
// pg_upgrade will refuse.
func newUpgradeFixture(ctx context.Context, t *testing.T, sourceSQL ...string) *upgradeFixture {
	t.Helper()

	dockerClient, err := client.New(client.FromEnv)
	require.NoError(t, err)

	image := pgUpgradeImage()
	if err := ensureUpgradeImage(ctx, dockerClient, image); err != nil {
		skipUnlessRequired(t, "upgrade image %s is unavailable: %v", image, err)
	}

	appConfig := shortPathAppConfig(t)
	require.NoError(t, os.MkdirAll(appConfig.CloneDir(), 0755))

	fixture := &upgradeFixture{
		appConfig: appConfig,
		state: UpgradeState{
			OldVersion:  upgradeSourceMajor,
			NewVersion:  upgradeTargetMajor,
			OldImage:    "postgresai/extended-postgres:" + upgradeSourceMajor,
			TargetImage: "postgresai/extended-postgres:" + upgradeTargetMajor,
			SnapshotID:  "snapshot",
		},
		image:   image,
		docker:  dockerClient,
		hostDir: t.TempDir(),
	}

	fixture.prepareUser(ctx, t)

	statements := append([]string{"create table upgrade_probe as select 42 as answer"}, sourceSQL...)
	psql := make([]string, 0, len(statements))

	for _, statement := range statements {
		// One invocation per statement: psql wraps several -c options in a single transaction,
		// and CREATE DATABASE cannot run inside one.
		psql = append(psql, fmt.Sprintf(`"${bin}"/psql -h /tmp -U dblab -d postgres -c '%s'`, statement))
	}

	require.Equal(t, 0, fixture.runShell(ctx, t, "source", fmt.Sprintf(`
set -eu
bin=/usr/lib/postgresql/%[1]s/bin
"${bin}"/initdb --pgdata "%[2]s" --encoding UTF8 --lc-collate C --lc-ctype C
"${bin}"/pg_ctl -D "%[2]s" -o "-k /tmp -c listen_addresses=''" -w start
%[3]s
"${bin}"/pg_ctl -D "%[2]s" -w -m fast stop
`, upgradeSourceMajor, appConfig.DataDir(), strings.Join(psql, "\n"))),
		"the source cluster must be created and cleanly shut down")

	provisioner := &Provisioner{runner: runners.NewLocalRunner(false)}

	owner, err := dataDirOwner(provisioner.runner, appConfig.DataDir())
	require.NoError(t, err)
	require.NoError(t, provisioner.prepareUpgradeWorkspace(appConfig, owner, fixture.state))

	return fixture
}

// prepareUser gives the image a passwd entry for the user that owns the clone directory. initdb
// refuses to start without one ("could not look up effective user ID"), and the alternative -
// handing the directory to the image's own postgres account - would need root for a chown. This
// also keeps the run faithful to the engine, which passes whoever owns the data directory to
// --user rather than a fixed account.
func (f *upgradeFixture) prepareUser(ctx context.Context, t *testing.T) {
	t.Helper()

	passwdPath := f.passwdPath()

	require.Equal(t, 0, f.runShell(ctx, t, "passwd", "cat /etc/passwd > "+passwdPath),
		"the image passwd file must be readable")

	// This is also the check that the daemon shares a filesystem with this process. The container
	// just wrote the file into a bind-mounted directory; if it is not here, the mount resolved
	// against the daemon's own filesystem, as it does when the daemon runs in a separate container
	// (docker:dind). These tests read a data directory that containers write, so they cannot run
	// there at all - and skipping says so instead of failing on a puzzling missing file.
	if _, err := os.Stat(passwdPath); err != nil {
		skipUnlessRequired(t, "the docker daemon does not share this filesystem, so a clone directory "+
			"written by a container is invisible here: %v", err)
	}

	file, err := os.OpenFile(passwdPath, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)

	defer file.Close()

	_, err = fmt.Fprintf(file, "dblab:x:%d:%d::%s:/bin/sh\n", os.Getuid(), os.Getgid(), f.appConfig.CloneDir())
	require.NoError(t, err)

	f.passwdReady = true
}

// upgrade runs the image the way the engine runs it and returns the exit code.
func (f *upgradeFixture) upgrade(ctx context.Context, t *testing.T, req UpgradeRequest) int {
	t.Helper()

	env := upgradeContainerEnv(f.appConfig.CloneDir(), f.appConfig.Pool.DataSubDir, f.state, req)

	return f.run(ctx, t, upgradeContainerName(f.appConfig.CloneName), &container.Config{
		Image:      f.image,
		Env:        append(env, "HOME="+f.appConfig.CloneDir()),
		User:       containerUser(),
		WorkingDir: f.appConfig.CloneDir(),
	})
}

// runShell executes a script in the upgrade image, which carries the binaries of both majors.
func (f *upgradeFixture) runShell(ctx context.Context, t *testing.T, name, script string) int {
	t.Helper()

	return f.run(ctx, t, "dblab_upgrade_test_"+name+"_"+f.appConfig.CloneName, &container.Config{
		Image:      f.image,
		Entrypoint: []string{"sh", "-c"},
		Cmd:        []string{script},
		User:       containerUser(),
		WorkingDir: f.appConfig.CloneDir(),
	})
}

func (f *upgradeFixture) run(ctx context.Context, t *testing.T, name string, cfg *container.Config) int {
	t.Helper()

	created, err := f.docker.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:           cfg,
		HostConfig:       &container.HostConfig{Binds: f.binds()},
		NetworkingConfig: &network.NetworkingConfig{},
		Name:             name,
	})
	require.NoError(t, err)

	defer func() {
		if _, err := f.docker.ContainerRemove(context.Background(), created.ID,
			client.ContainerRemoveOptions{Force: true}); err != nil {
			t.Logf("failed to remove container %s: %v", name, err)
		}
	}()

	// The wait has to be requested before the start and with this condition: "not-running" is
	// already true for a container that has been created but never started, so it would return
	// zero immediately and the deferred removal would kill the run.
	waiting := f.docker.ContainerWait(ctx, created.ID, client.ContainerWaitOptions{
		Condition: container.WaitConditionNextExit,
	})

	_, err = f.docker.ContainerStart(ctx, created.ID, client.ContainerStartOptions{})
	require.NoError(t, err)

	select {
	case err := <-waiting.Error:
		require.NoError(t, err)

	case result := <-waiting.Result:
		if result.StatusCode != 0 {
			t.Logf("container %s exited with %d:\n%s", name, result.StatusCode, f.logs(ctx, t, created.ID))
		}

		return int(result.StatusCode)
	}

	return 0
}

func (f *upgradeFixture) binds() []string {
	binds := []string{
		f.appConfig.CloneDir() + ":" + f.appConfig.CloneDir(),
		f.hostDir + ":" + f.hostDir,
	}

	if f.passwdReady {
		binds = append(binds, f.passwdPath()+":/etc/passwd:ro")
	}

	return binds
}

func (f *upgradeFixture) passwdPath() string {
	return path.Join(f.hostDir, "passwd")
}

// logs returns the container output plus the script's own log, which is where pg_upgrade's
// complaint ends up.
func (f *upgradeFixture) logs(ctx context.Context, t *testing.T, containerID string) string {
	t.Helper()

	out := strings.Builder{}

	if reader, err := f.docker.ContainerLogs(ctx, containerID,
		client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}); err == nil {
		defer reader.Close()

		if content, err := io.ReadAll(reader); err == nil {
			out.Write(content)
		}
	}

	if content, err := os.ReadFile(upgradeLogPath(f.appConfig.CloneDir())); err == nil {
		out.WriteString("\n--- pg_upgrade_clone.log ---\n")
		out.Write(content)
	}

	return out.String()
}

// assertProbeRow starts the given cluster with the given major and fails unless the row written
// before the upgrade is still readable.
func (f *upgradeFixture) assertProbeRow(ctx context.Context, t *testing.T, major, dataDir string) {
	t.Helper()

	code := f.runShell(ctx, t, "verify"+major, fmt.Sprintf(`
set -eu
bin=/usr/lib/postgresql/%[1]s/bin
"${bin}"/pg_ctl -D "%[2]s" -o "-k /tmp -c listen_addresses=''" -w start
"${bin}"/psql -h /tmp -U dblab -d postgres -tAc "%[3]s" | grep -qx 42
"${bin}"/pg_ctl -D "%[2]s" -w -m fast stop
`, major, dataDir, upgradeProbeQuery))

	assert.Equal(t, 0, code, "the cluster in %s must be startable and still hold the probe row", dataDir)
}

func containerUser() string {
	return fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
}

// shortPathAppConfig keeps the clone directory short. A Unix socket path may not exceed 107 bytes,
// and pg_upgrade puts its own sockets in the working directory it is given - the clone directory -
// so a t.TempDir() carrying the test name is enough to break the run for reasons that have nothing
// to do with what is under test.
func shortPathAppConfig(t *testing.T) *resources.AppConfig {
	t.Helper()

	mountDir, err := os.MkdirTemp("", "dblab-upg-")
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := os.RemoveAll(mountDir); err != nil {
			t.Logf("failed to remove %s: %v", mountDir, err)
		}
	})

	pool := &resources.Pool{MountDir: mountDir, PoolDirName: "p", DataSubDir: "data"}

	return &resources.AppConfig{CloneName: "c1", Branch: "main", Pool: pool}
}

func upgradeRequest(t *testing.T) UpgradeRequest {
	t.Helper()

	target, err := strconv.Atoi(upgradeTargetMajor)
	require.NoError(t, err)

	return UpgradeRequest{
		TargetVersion: target,
		TargetImage:   "postgresai/extended-postgres:" + upgradeTargetMajor,
		Encoding:      "UTF8",
		LCCollate:     "C",
		LCCtype:       "C",
		DataChecksums: "off",
	}
}

func TestUpgradeScriptConvertsTheClusterInPlace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), upgradeRunTimeout)
	defer cancel()

	fixture := newUpgradeFixture(ctx, t)

	require.Equal(t, 0, fixture.upgrade(ctx, t, upgradeRequest(t)))

	facts := readDataDirFacts(fixture.appConfig, fixture.state)

	assert.True(t, facts.NewClusterReady, "the converted cluster carries the target major")
	assert.True(t, facts.OldClusterDisabled, "pg_upgrade renames global/pg_control away once linking starts")
	assert.False(t, facts.SwapCompleted, "the swap belongs to the engine, not to the script")
	assert.True(t, facts.ConversionDone,
		"a run that returned 0 must record the completion marker; without it nothing distinguishes "+
			"this directory from one whose transfer was killed part-way through")
	assert.Equal(t, recoveryComplete, classifyOutcome(facts))

	fixture.assertProbeRow(ctx, t, upgradeTargetMajor, newDataDir(fixture.appConfig))
}

// TestPostSwapStartFailureLeavesTheCloneRecoverable covers the window that used to cost a clone its
// dataset: the conversion succeeded and the engine promoted the new data directory, then starting
// the clone container failed. Nothing has been cleared at that point, so the next engine start has
// to see an upgrade that is still pending and finish it - if it instead skipped the clone, the
// restore path would drop a clone with no container and the cleanup pass would destroy a dataset
// holding a completed upgrade.
func TestPostSwapStartFailureLeavesTheCloneRecoverable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), upgradeRunTimeout)
	defer cancel()

	fixture := newUpgradeFixture(ctx, t)

	require.Equal(t, 0, fixture.upgrade(ctx, t, upgradeRequest(t)))
	require.NoError(t, swapDataDirs(fixture.appConfig))

	// postgres.Start fails here. It is the only step completeUpgrade takes after the swap that can
	// fail on a healthy data directory, and it clears no artifact of its own.
	assert.True(t, hasPendingUpgrade(fixture.appConfig.CloneDir()),
		"the upgrade state has to outlive the failure, because it is what selects the clone for recovery")

	facts := readDataDirFacts(fixture.appConfig, fixture.state)

	assert.True(t, facts.SwapCompleted, "the clone data directory now holds the target major")
	assert.Equal(t, recoveryComplete, classifyOutcome(facts), "recovery has to finish the upgrade, not undo it")

	fixture.assertProbeRow(ctx, t, upgradeTargetMajor, fixture.appConfig.DataDir())
}

// TestUpgradeScriptCheckFailureKeepsTheOldClusterStartable covers the failure that made the
// classification wrong: pg_upgrade --check rejects the pair, so nothing is converted, yet initdb
// has already written the target major into <data>_new.
func TestUpgradeScriptCheckFailureKeepsTheOldClusterStartable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), upgradeRunTimeout)
	defer cancel()

	// A database that refuses connections is one of the cluster states pg_upgrade rejects outright,
	// and it is rejected in the check phase - after the new cluster has been initdb'd.
	fixture := newUpgradeFixture(ctx, t,
		"create database blocked",
		"alter database blocked allow_connections false")

	require.Equal(t, exitPreTransfer, fixture.upgrade(ctx, t, upgradeRequest(t)))

	facts := readDataDirFacts(fixture.appConfig, fixture.state)

	assert.True(t, facts.NewClusterReady,
		"initdb runs before the check, so the target major is on disk even though nothing was converted")
	assert.False(t, facts.OldClusterDisabled, "a failed check leaves the old cluster startable")
	assert.False(t, facts.SwapCompleted)
	assert.False(t, facts.ConversionDone, "the script records completion only after pg_upgrade returns 0")
	assert.FileExists(t, path.Join(newDataDir(fixture.appConfig), "PG_VERSION"))

	// The whole point: these facts must not read as a finished upgrade.
	assert.Equal(t, recoveryRollback, classifyOutcome(facts))

	fixture.assertProbeRow(ctx, t, upgradeSourceMajor, fixture.appConfig.DataDir())
}

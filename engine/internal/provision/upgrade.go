/*
2026 © Postgres.ai
*/

package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/databases/postgres"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/databases/postgres/pgconfig"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/docker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const (
	// Exit codes of scripts/pg_upgrade_clone.sh. They report what the script believed happened
	// and are used only to explain a failure: the recovery decision comes from the data
	// directory, which cannot be stale or unwritable the way a reported code can.
	exitPreTransfer  = 10
	exitPostTransfer = 20

	// upgradeDirName holds everything the upgrade writes outside the data directory. It is a
	// directory of its own so that a single chown hands the whole working area to the user the
	// upgrade container runs as.
	upgradeDirName = "upgrade"

	upgradeStateFileName = "state.json"
	upgradeStageFileName = "stage"
	upgradeLogsDirName   = "logs"
	upgradeLogFileName   = "pg_upgrade_clone.log"

	// upgradeDoneMarkerName is written by scripts/pg_upgrade_clone.sh into the new data directory
	// once pg_upgrade has returned 0, and is the only proof this engine accepts that the
	// conversion finished. It lives inside the cluster directory so that it survives swapDataDirs.
	// Keep the name in step with DONE_MARKER_NAME in that script.
	upgradeDoneMarkerName = ".dblab_upgrade_done"

	// maxMajorJump is how many majors back the upgrade image carries binaries for.
	maxMajorJump = 4

	// minSourceMajor is the oldest major the upgrade image can convert from. Dockerfile.pg-upgrade
	// clamps the lower end of its OLD_MAJORS range at 12, so an older source has no binaries to
	// be started with however small the jump is.
	minSourceMajor = 12

	// stopPostgresTimeout bounds the graceful shutdown that has to precede pg_upgrade.
	stopPostgresTimeout = 300

	// defaultUpgradeTimeout bounds a single run of the upgrade container. Everything else in the
	// flow is already bounded - stopPostgresTimeout, the pre-flight queries, the postgres.Start
	// wait loop - and the conversion must be too: a hung pg_upgrade would hold the clone in
	// UPGRADING, a status the idle sweeper exempts and destroy and reset refuse, so nothing short
	// of an engine restart could clear it. The budget is generous because pg_upgrade --link is
	// dominated by dumping and restoring the schema, which a pathological catalog can stretch to
	// hours; provision.pgUpgradeTimeout raises it for the clusters that need more.
	defaultUpgradeTimeout = 3 * time.Hour

	// defaultUpgradePullTimeout bounds a single image pull made for an upgrade. The upgrade image
	// carries the target major plus the server packages of the four preceding ones, so a cold
	// instance downloads several gigabytes here.
	//
	// This is deliberately a separate budget from defaultUpgradeTimeout rather than a share of it:
	// a slow pull must not eat into the time pg_upgrade gets once the clone is already stopped,
	// which is the one phase that cannot be retried cheaply. provision.pgUpgradePullTimeout raises
	// it for instances on slow links or far from their registry.
	defaultUpgradePullTimeout = time.Hour

	// logTailLines bounds the pg_upgrade log excerpt carried in a clone status message. It is
	// deliberately separate from the container-log tail in the tools package: the two bound
	// unrelated outputs and happen to agree today.
	logTailLines = 20
)

// upgradeStage marks how far scripts/pg_upgrade_clone.sh got. It is a diagnostic breadcrumb
// only: recovery decisions come from the data directory itself, because a stage file can be
// missing, stale, or unwritable while the cluster on disk is unambiguous.
type upgradeStage string

const (
	stagePrepare  upgradeStage = "prepare"
	stageInitdb   upgradeStage = "initdb"
	stageCheck    upgradeStage = "check"
	stageTransfer upgradeStage = "transfer"
	stageDone     upgradeStage = "done"
	stageUnknown  upgradeStage = "unknown"
)

// recoveryAction is what the engine must do about a data directory whose upgrade did not finish.
type recoveryAction string

const (
	// recoveryRollback restores the clone in place: the old cluster is still startable.
	recoveryRollback recoveryAction = "rollback"

	// recoveryReprovision re-creates the clone from its origin snapshot. pg_upgrade has already
	// disabled the old cluster, so it must never be started again.
	recoveryReprovision recoveryAction = "reprovision"

	// recoveryComplete finishes an upgrade whose data directory was already swapped.
	recoveryComplete recoveryAction = "complete"
)

// UpgradeOutcome tells the cloning layer which status and message a finished upgrade deserves.
type UpgradeOutcome string

const (
	// UpgradeSucceeded means the clone now runs the target major.
	UpgradeSucceeded UpgradeOutcome = "succeeded"

	// UpgradeRolledBack means the upgrade did not convert anything and the clone is running
	// again on its original major, with nothing lost.
	UpgradeRolledBack UpgradeOutcome = "rolled_back"

	// UpgradeRequiresReprovision means the old cluster can no longer be started and the clone
	// has to be re-created from its origin snapshot. The provisioner does not do that itself:
	// re-provisioning has to go through the cloning layer, which knows about dependent
	// snapshots and bumps the clone revision before destroying the dataset.
	UpgradeRequiresReprovision UpgradeOutcome = "requires_reprovision"

	// UpgradeReprovisioned is the terminal form of UpgradeRequiresReprovision, set by the
	// cloning layer once the clone has actually been rebuilt.
	UpgradeReprovisioned UpgradeOutcome = "reprovisioned"

	// UpgradeUnchanged means recovery found nothing in flight: the clone is left exactly as
	// PostgreSQL last wrote it, either because the upgrade never reached it or because it had
	// already settled when the engine stopped.
	UpgradeUnchanged UpgradeOutcome = "unchanged"
)

// UpgradeRequest carries everything the upgrade container needs. The collation fields are read
// from the running clone before it is stopped: initdb of the new cluster has to match the old
// cluster exactly or pg_upgrade refuses the pair.
type UpgradeRequest struct {
	TargetVersion    int
	TargetImage      string
	Encoding         string
	LCCollate        string
	LCCtype          string
	LocaleProvider   string
	ICULocale        string
	DataChecksums    string
	OldServerOptions string
}

// UpgradeState records what an in-flight upgrade is doing, so an engine restart can tell what to
// finish or undo. It is removed once the upgrade settles either way.
type UpgradeState struct {
	OldVersion  string `json:"oldVersion"`
	NewVersion  string `json:"newVersion"`
	OldImage    string `json:"oldImage"`
	TargetImage string `json:"targetImage"`
	SnapshotID  string `json:"snapshotID"`
}

// UpgradeResult describes how an upgrade ended.
type UpgradeResult struct {
	Outcome UpgradeOutcome
	// Version is the major the clone runs, or would run once re-provisioned.
	Version string
	// PreviousVersion is the major the clone ran before the call.
	PreviousVersion string
	// Image is the image the clone container runs, or would run once re-provisioned.
	Image string
	// SnapshotID is the origin snapshot a reprovision has to rebuild from.
	SnapshotID string
	// Cause is the original failure when the outcome is not UpgradeSucceeded.
	Cause string
	// LogPath points at the pg_upgrade log inside the clone dataset. Empty when the dataset is
	// about to be re-created, because the log goes with it.
	LogPath string
	// LogTail holds the last lines of that log, captured before any dataset destruction.
	LogTail string
	// Recoverable is set on a failure that left the clone without a container but with its upgrade
	// state still on disk, which is what makes the next engine start pick the clone up again. The
	// cloning layer needs it to tell such a clone from one nothing can be done about any more,
	// because the status it records for the latter costs the clone its dataset.
	Recoverable bool
}

// dataDirFacts is what the clone's data directory says about an interrupted upgrade. These are
// the authoritative inputs to a recovery decision: a stage file may be missing or unwritable,
// but pg_control is the same marker PostgreSQL itself uses.
type dataDirFacts struct {
	// SwapCompleted is true when the data directory already holds the target major.
	SwapCompleted bool
	// NewClusterReady is true when the new data directory holds the target major. The script
	// initdb's that cluster before pg_upgrade runs, so on its own this says only that the upgrade
	// got as far as creating the target; it means "converted" only together with
	// OldClusterDisabled.
	NewClusterReady bool
	// OldClusterDisabled is true when pg_upgrade has renamed global/pg_control away. That happens
	// when linking STARTS, not when it finishes, so this says only that the old cluster is no
	// longer startable - never that there is a finished cluster to promote in its place.
	OldClusterDisabled bool
	// ConversionDone is true when the upgrade script recorded that pg_upgrade returned 0. It is
	// the only fact that distinguishes a finished conversion from one killed part-way through,
	// because every other marker here is already in place before the transfer begins.
	ConversionDone bool
}

// UpgradeSession performs an in-place major upgrade of a clone. It returns an error only when
// the clone is left without a running container; anything the engine could recover from is
// reported through the result, because the caller only needs to explain what happened.
func (p *Provisioner) UpgradeSession(
	session *resources.Session, clone *models.Clone, req UpgradeRequest) (UpgradeResult, error) {
	appConfig, err := p.upgradeAppConfig(session, clone)
	if err != nil {
		return abortedUpgrade(err.Error()), nil
	}

	currentVersion, err := tools.DetectPGVersion(appConfig.DataDir())
	if err != nil {
		return abortedUpgrade(fmt.Sprintf("failed to detect the current PostgreSQL version: %s", err)), nil
	}

	state := UpgradeState{
		OldVersion:  formatPGVersion(currentVersion),
		NewVersion:  strconv.Itoa(req.TargetVersion),
		OldImage:    resolveCloneImage(p.config.DockerImage, clone.DockerImage),
		TargetImage: req.TargetImage,
		SnapshotID:  clone.Snapshot.ID,
	}

	if err := validateUpgradeTarget(currentVersion, req.TargetVersion); err != nil {
		return abortedUpgradeFrom(state, err.Error()), nil
	}

	// Both images are pulled while the clone still serves traffic: a multi-gigabyte download
	// inside the downtime window is indistinguishable from a hang. A pull failure here leaves
	// the clone untouched, so it is an aborted upgrade rather than a broken clone.
	for _, image := range []string{p.config.PgUpgradeImage, req.TargetImage} {
		if err := p.prepareUpgradeImage(image); err != nil {
			return abortedUpgradeFrom(state, err.Error()), nil
		}
	}

	owner, err := dataDirOwner(p.runner, appConfig.DataDir())
	if err != nil {
		return abortedUpgradeFrom(state, err.Error()), nil
	}

	if err := p.prepareUpgradeWorkspace(appConfig, owner, state); err != nil {
		return abortPreparedUpgrade(appConfig, state, err.Error()), nil
	}

	// From here on the clone is stopped, so every path has to end with either a running clone
	// or an explicit reprovision request.
	if err := p.stopCloneForUpgrade(appConfig); err != nil {
		return abortPreparedUpgrade(appConfig, state, err.Error()), nil
	}

	containerName := upgradeContainerName(appConfig.CloneName)

	// An engine that died during an earlier attempt leaves the container behind, and docker run
	// refuses a name that is already taken. Recovery removes it for the same reason.
	if _, err := docker.RemoveContainer(p.runner, containerName); err != nil {
		log.Dbg("no stale upgrade container to remove for clone", appConfig.CloneName, err)
	}

	// Deliberately NOT derived from p.ctx. That context ends when the engine is asked to shut
	// down, and cancelling this one force-removes the upgrade container - so an ordinary
	// `docker restart` of the engine would kill pg_upgrade mid-transfer and leave the clone to be
	// rebuilt from its snapshot. The container is detached precisely so it can outlive the engine:
	// RecoverUpgrade removes and reclassifies it on the next start. Only the upgrade budget itself
	// may end this wait.
	ctx, cancel := context.WithTimeout(context.Background(), p.upgradeTimeout())
	defer cancel()

	exitCode, runErr := docker.RunUpgradeContainer(ctx, p.dockerClient, p.runner, appConfig, docker.UpgradeContainerConfig{
		Image: p.config.PgUpgradeImage,
		Name:  containerName,
		User:  owner,
		Env:   upgradeContainerEnv(appConfig.CloneDir(), appConfig.Pool.DataSubDir, state, req),
	})

	cause := describeExitCode(exitCode)
	if runErr != nil {
		cause = fmt.Sprintf("failed to run the upgrade container: %s", runErr)
		exitCode = -1
	}

	if exitCode == 0 {
		result, err := p.completeUpgrade(appConfig, state)

		return markRecoverable(appConfig, result, err)
	}

	// A run that timed out is classified exactly like any other failure: the container is gone by
	// now, and the data directory says whether the old cluster can still be started.
	result, err := p.recover(classifyOutcome(readDataDirFacts(appConfig, state)), appConfig, state, cause)

	return markRecoverable(appConfig, result, err)
}

// upgradeTimeout is how long a single upgrade container may run.
func (p *Provisioner) upgradeTimeout() time.Duration {
	if p.config.PgUpgradeTimeout > 0 {
		return p.config.PgUpgradeTimeout
	}

	return defaultUpgradeTimeout
}

// upgradePullTimeout is how long a single image pull made for an upgrade may take.
func (p *Provisioner) upgradePullTimeout() time.Duration {
	if p.config.PgUpgradePullTimeout > 0 {
		return p.config.PgUpgradePullTimeout
	}

	return defaultUpgradePullTimeout
}

// prepareUpgradeImage makes sure an image is present locally, bounding the pull it may need.
// p.ctx lives as long as the engine, so an unreachable or stalled registry would otherwise hold
// the clone in UPGRADING indefinitely - a status the idle sweeper exempts and destroy and reset
// refuse, which is the same dead end upgradeTimeout exists to prevent, one phase earlier.
func (p *Provisioner) prepareUpgradeImage(image string) error {
	timeout := p.upgradePullTimeout()

	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	err := docker.PrepareImage(ctx, p.dockerClient, image)
	if err == nil {
		return nil
	}

	// The docker client reports a cancelled pull as a transport error, so the deadline has to be
	// read from the context to be named as one. A cancelled parent means the engine is shutting
	// down and is left to the generic message.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timed out after %s preparing docker image %s", timeout, image)
	}

	return fmt.Errorf("cannot prepare docker image %s: %s", image, err)
}

// RecoverUpgrade settles a clone that was still upgrading when the engine stopped. It runs
// before the restore path filters clones, so a clone whose container is gone is brought back
// rather than swept into dataset destruction.
func (p *Provisioner) RecoverUpgrade(session *resources.Session, clone *models.Clone) (UpgradeResult, error) {
	appConfig, err := p.upgradeAppConfig(session, clone)
	if err != nil {
		return UpgradeResult{}, err
	}

	state, err := readUpgradeState(appConfig.CloneDir())
	if err != nil {
		// The state file exists for exactly as long as the clone is at risk: it is written before
		// the clone is stopped and removed once the upgrade has settled. Its absence therefore
		// means there is nothing to finish or undo. Reporting a failure here instead would put a
		// running, untouched clone into FATAL, and a FATAL clone loses its dataset to the next
		// cleanup pass.
		log.Dbg("no upgrade state to recover for clone", clone.ID, err)

		return settledClone(appConfig), nil
	}

	// The upgrade container is detached, so it outlives an engine that was killed rather than
	// shut down. Removing it before touching the dataset stops a live pg_upgrade from writing
	// into a directory that is about to be restarted or destroyed.
	if _, err := docker.RemoveContainer(p.runner, upgradeContainerName(appConfig.CloneName)); err != nil {
		log.Dbg("no upgrade container to remove for clone", clone.ID, err)
	}

	facts := readDataDirFacts(appConfig, state)
	action := classifyOutcome(facts)

	log.Msg(fmt.Sprintf("Recovering clone %s after an interrupted upgrade: %s (stage %q, swapped=%t, old cluster disabled=%t)",
		clone.ID, action, readUpgradeStage(appConfig.CloneDir()), facts.SwapCompleted, facts.OldClusterDisabled))

	if action == recoveryComplete {
		result, err := p.completeUpgrade(appConfig, state)

		return markRecoverable(appConfig, result, err)
	}

	result, err := p.recover(action, appConfig, state, "upgrade was interrupted by an engine restart")

	return markRecoverable(appConfig, result, err)
}

// HasPendingUpgrade reports whether the clone directory still carries an upgrade that has not
// settled. It is what selects the clones to recover on engine start, because the clone status
// cannot do it: an upgrade that fails after the clone was stopped records a status of its own, and
// keying recovery on that status is how a clone whose data is already converted ends up swept into
// dataset destruction instead of being brought back up.
func (p *Provisioner) HasPendingUpgrade(session *resources.Session, clone *models.Clone) bool {
	appConfig, err := p.upgradeAppConfig(session, clone)
	if err != nil {
		log.Dbg("cannot inspect the upgrade state of clone", clone.ID, err)

		return false
	}

	return hasPendingUpgrade(appConfig.CloneDir())
}

// hasPendingUpgrade reads the same file RecoverUpgrade reads. It exists for exactly as long as the
// clone is at risk: written before the clone is stopped, removed once the upgrade has settled.
func hasPendingUpgrade(cloneDir string) bool {
	_, err := os.Stat(path.Join(upgradeDirPath(cloneDir), upgradeStateFileName))

	return err == nil
}

// markRecoverable tags a failure the next engine start can still do something about. Both failure
// paths leave the clone without a container, and which of them the cloning layer may treat as
// terminal depends on nothing but whether the upgrade state survived.
func markRecoverable(appConfig *resources.AppConfig, result UpgradeResult, err error) (UpgradeResult, error) {
	if err != nil {
		result.Recoverable = hasPendingUpgrade(appConfig.CloneDir())
	}

	return result, err
}

// upgradeAppConfig rebuilds the container configuration of an existing clone.
func (p *Provisioner) upgradeAppConfig(
	session *resources.Session, clone *models.Clone) (*resources.AppConfig, error) {
	fsm, err := p.pm.GetFSManager(session.Pool)
	if err != nil {
		return nil, fmt.Errorf("failed to find filesystem manager of this session: %w", err)
	}

	appConfig := p.getAppConfig(fsm.Pool(), clone.Branch, clone.ID, clone.Revision, session.Port)
	appConfig.SetExtraConf(session.ExtraConfig)

	return appConfig, nil
}

// prepareUpgradeWorkspace creates everything the upgrade container writes into and hands it to
// the user that container runs as.
//
// The clone directory belongs to the engine OS user and the container runs as the data directory
// owner, which is a different account in the standard deployment. The container therefore cannot
// create or rename anything inside the clone directory - not even the new data directory - so
// the engine pre-creates the empty target and performs the swap itself afterwards.
func (p *Provisioner) prepareUpgradeWorkspace(appConfig *resources.AppConfig, owner string, state UpgradeState) error {
	upgradeDir := upgradeDirPath(appConfig.CloneDir())

	if err := os.MkdirAll(path.Join(upgradeDir, upgradeLogsDirName), 0755); err != nil {
		return fmt.Errorf("failed to create the upgrade directory: %w", err)
	}

	if err := writeUpgradeState(appConfig.CloneDir(), state); err != nil {
		return err
	}

	if err := writeUpgradeStage(appConfig.CloneDir(), stagePrepare); err != nil {
		return err
	}

	// A leftover from an earlier attempt would make initdb refuse a non-empty directory.
	if err := os.RemoveAll(newDataDir(appConfig)); err != nil {
		return fmt.Errorf("failed to clear a stale new data directory: %w", err)
	}

	// Tidy away a marker left in the data directory by an earlier upgrade of this clone. Nothing
	// depends on this: readDataDirFacts reads the marker only from the new cluster, which is
	// re-created just below. It is best-effort housekeeping so that a stray dot-file does not end
	// up in a snapshot taken later.
	if err := os.Remove(path.Join(appConfig.DataDir(), upgradeDoneMarkerName)); err != nil && !os.IsNotExist(err) {
		log.Dbg("could not clear a stale upgrade completion marker:", err)
	}

	// initdb insists on 0700 and accepts an empty directory it already owns.
	if err := os.MkdirAll(newDataDir(appConfig), 0700); err != nil {
		return fmt.Errorf("failed to create the new data directory: %w", err)
	}

	// Hand over the least that lets the container work: its log directory and the cluster it
	// builds, plus the stage file it reports progress through.
	//
	// The upgrade directory itself stays with the engine. It holds state.json, which names the
	// images RecoverUpgrade starts the clone on, and a directory the container could write would
	// let a clone's own DB superuser - the default for an ephemeral clone user - replace that file
	// and choose an image the engine then runs. The stage file is pre-created above and chowned
	// here instead, so the container can overwrite it in place without ever being able to create
	// or replace a sibling.
	handOver := []string{
		path.Join(upgradeDir, upgradeLogsDirName),
		path.Join(upgradeDir, upgradeStageFileName),
		newDataDir(appConfig),
	}

	for _, target := range handOver {
		if _, err := p.runner.Run(fmt.Sprintf("chown -R %s %s", owner, runners.Quote(target)), false); err != nil {
			return fmt.Errorf("failed to hand %s to %s: %w", target, owner, err)
		}
	}

	return nil
}

// swapDataDirs promotes the converted cluster to be the clone's data directory. It is idempotent
// so that a crash between the two renames can be finished on restart, and it runs in the engine
// because renaming inside the clone directory needs an access the upgrade container lacks.
func swapDataDirs(appConfig *resources.AppConfig) error {
	dataDir, newDir, backupDir := appConfig.DataDir(), newDataDir(appConfig), backupDataDir(appConfig)

	if _, err := os.Stat(newDir); err != nil {
		// Already promoted by an earlier attempt.
		return nil
	}

	if _, err := os.Stat(dataDir); err == nil {
		if err := os.RemoveAll(backupDir); err != nil {
			return fmt.Errorf("failed to clear a stale pre-upgrade data directory: %w", err)
		}

		if err := os.Rename(dataDir, backupDir); err != nil {
			return fmt.Errorf("failed to move the pre-upgrade data directory aside: %w", err)
		}
	}

	if err := os.Rename(newDir, dataDir); err != nil {
		return fmt.Errorf("failed to promote the upgraded data directory: %w", err)
	}

	return nil
}

// stopCloneForUpgrade shuts Postgres down cleanly and then drops the container. pg_upgrade
// refuses a source cluster that was not shut down cleanly, and postgres.Stop is a forced
// container removal, so the order matters. A failure here leaves the container in place, which
// is why the caller treats it as an aborted upgrade and never tries to start a second one.
func (p *Provisioner) stopCloneForUpgrade(appConfig *resources.AppConfig) error {
	if err := tools.StopPostgres(
		p.ctx, p.dockerClient, appConfig.CloneName, appConfig.DataDir(), stopPostgresTimeout); err != nil {
		return fmt.Errorf("failed to stop Postgres gracefully, the clone keeps running: %w", err)
	}

	port := strconv.FormatUint(uint64(appConfig.Port), 10)
	if err := postgres.Stop(p.runner, appConfig.Pool, appConfig.CloneName, port); err != nil {
		return fmt.Errorf("failed to remove the clone container: %w", err)
	}

	return nil
}

// completeUpgrade brings the clone up on the converted data directory.
func (p *Provisioner) completeUpgrade(appConfig *resources.AppConfig, state UpgradeState) (UpgradeResult, error) {
	if err := swapDataDirs(appConfig); err != nil {
		return UpgradeResult{}, err
	}

	// The new data directory came straight from initdb and carries none of the DBLab-managed
	// configuration, so the clone would boot unreachable without this.
	if _, err := pgconfig.NewCorrector(appConfig.DataDir()); err != nil {
		return UpgradeResult{}, fmt.Errorf("failed to apply Database Lab configuration to the upgraded data directory: %w", err)
	}

	appConfig.DockerImage = state.TargetImage

	if err := postgres.Start(p.runner, appConfig); err != nil {
		return UpgradeResult{}, fmt.Errorf("failed to start the upgraded clone: %w", err)
	}

	// The marker describes one conversion and must not outlive it: it has just travelled into the
	// data directory with the swap, and a copy left there would make the NEXT upgrade of this
	// clone look finished the moment it disabled the old cluster.
	if err := os.Remove(path.Join(appConfig.DataDir(), upgradeDoneMarkerName)); err != nil && !os.IsNotExist(err) {
		log.Err("failed to remove the upgrade completion marker:", err)
	}

	// The previous cluster is hard-linked to the new one, so removing it reclaims only the
	// files the upgrade actually rewrote.
	if err := os.RemoveAll(backupDataDir(appConfig)); err != nil {
		log.Err("failed to remove the pre-upgrade data directory:", err)
	}

	clearUpgradeArtifacts(appConfig.CloneDir())

	return UpgradeResult{
		Outcome:         UpgradeSucceeded,
		Version:         state.NewVersion,
		PreviousVersion: state.OldVersion,
		Image:           state.TargetImage,
		LogPath:         upgradeLogPath(appConfig.CloneDir()),
	}, nil
}

// recover undoes an upgrade that did not finish.
func (p *Provisioner) recover(
	action recoveryAction, appConfig *resources.AppConfig, state UpgradeState, cause string) (UpgradeResult, error) {
	result := UpgradeResult{
		Version:         state.OldVersion,
		PreviousVersion: state.OldVersion,
		Image:           state.OldImage,
		SnapshotID:      state.SnapshotID,
		Cause:           cause,
		LogTail:         readLogTail(appConfig.CloneDir()),
	}

	switch action {
	case recoveryComplete:
		// Reached when the swap completed but the script was killed before reporting success.
		return p.completeUpgrade(appConfig, state)

	case recoveryReprovision:
		// The dataset has to go, so the log goes with it; the tail above is what survives.
		// Re-provisioning itself belongs to the cloning layer, which handles dependent
		// snapshots and the revision bump.
		result.Outcome = UpgradeRequiresReprovision

		return result, nil
	}

	if err := os.RemoveAll(newDataDir(appConfig)); err != nil {
		log.Err("failed to remove the partially initialized data directory:", err)
	}

	// Recovery after an engine restart can find the clone container still running, because the
	// upgrade never got as far as stopping it. postgres.Start runs `docker run`, which fails on a
	// name that is already taken, so the leftover has to go first. On the ordinary failure path
	// the container is long gone and this does nothing.
	if err := p.stopCloneForUpgrade(appConfig); err != nil {
		log.Dbg("no clone container to remove before restarting it:", err)
	}

	appConfig.DockerImage = state.OldImage

	if err := postgres.Start(p.runner, appConfig); err != nil {
		return result, fmt.Errorf("%s; the clone could not be restarted on PostgreSQL %s either: %w",
			cause, state.OldVersion, err)
	}

	clearUpgradeArtifacts(appConfig.CloneDir())

	result.Outcome = UpgradeRolledBack
	result.LogPath = upgradeLogPath(appConfig.CloneDir())

	return result, nil
}

// abortedUpgrade reports an upgrade that never started. The clone was never stopped, so it is
// still running and only the reason has to reach the user.
func abortedUpgrade(cause string) UpgradeResult {
	return UpgradeResult{Outcome: UpgradeRolledBack, Cause: cause}
}

// abortPreparedUpgrade reports an upgrade that stopped after the workspace had been prepared but
// before anything was converted. The clone is still running, so the workspace goes with the
// attempt: a state file left behind marks the clone as mid-upgrade, and the next engine start
// would stop a healthy clone to recover an upgrade that never began. The empty new data directory
// goes too, because a snapshot taken meanwhile would otherwise carry it.
func abortPreparedUpgrade(appConfig *resources.AppConfig, state UpgradeState, cause string) UpgradeResult {
	if err := os.RemoveAll(newDataDir(appConfig)); err != nil {
		log.Err("failed to remove the prepared data directory:", err)
	}

	clearUpgradeArtifacts(appConfig.CloneDir())

	return abortedUpgradeFrom(state, cause)
}

func abortedUpgradeFrom(state UpgradeState, cause string) UpgradeResult {
	return UpgradeResult{
		Outcome:         UpgradeRolledBack,
		Version:         state.OldVersion,
		PreviousVersion: state.OldVersion,
		Image:           state.OldImage,
		Cause:           cause,
	}
}

// settledClone reports a clone that has no upgrade to recover. The version is read from the data
// directory rather than taken from the clone record, which is one step behind when the engine
// stopped between finishing an upgrade and persisting its result.
func settledClone(appConfig *resources.AppConfig) UpgradeResult {
	result := UpgradeResult{Outcome: UpgradeUnchanged}

	if version, err := tools.DetectPGVersion(appConfig.DataDir()); err == nil {
		result.Version = formatPGVersion(version)
	}

	return result
}

// readDataDirFacts inspects the clone data directory. Every marker here is written by PostgreSQL
// itself, which makes them trustworthy in ways a script breadcrumb is not - but a PG_VERSION
// dates an initdb rather than a conversion, so only pg_control tells the two apart.
func readDataDirFacts(appConfig *resources.AppConfig, state UpgradeState) dataDirFacts {
	facts := dataDirFacts{}

	if version, err := tools.DetectPGVersion(appConfig.DataDir()); err == nil {
		facts.SwapCompleted = formatPGVersion(version) == state.NewVersion
	}

	if version, err := tools.DetectPGVersion(newDataDir(appConfig)); err == nil {
		facts.NewClusterReady = formatPGVersion(version) == state.NewVersion
	}

	// pg_upgrade renames global/pg_control out of the way to stop the old cluster from being
	// started once linking has begun. Note "begun": PostgreSQL performs this rename immediately
	// BEFORE it transfers the relation files, so its absence dates the start of the transfer and
	// says nothing about whether that transfer ever finished.
	if _, err := os.Stat(path.Join(appConfig.DataDir(), "global", "pg_control")); err != nil {
		facts.OldClusterDisabled = true
	}

	// Which is why completion needs a marker of its own, and why it is read ONLY from the new
	// cluster. That directory is removed and re-created at the start of every attempt, so a marker
	// found there can only have been written by this conversion. The data directory is not
	// consulted: after a swap SwapCompleted already decides the outcome without it, so the only
	// case where reading it could change anything is the un-swapped one - where the data directory
	// is still the OLD cluster, is owned by the clone's postgres user, and a leftover or a file
	// planted through the running clone would vouch for a transfer that never finished.
	if _, err := os.Stat(path.Join(newDataDir(appConfig), upgradeDoneMarkerName)); err == nil {
		facts.ConversionDone = true
	}

	return facts
}

// describeExitCode turns the script's exit code into something a user can act on. It explains
// the failure only; what the engine does about it is decided from the data directory.
func describeExitCode(exitCode int) string {
	switch exitCode {
	case exitPreTransfer:
		return fmt.Sprintf("pg_upgrade failed with exit code %d before converting anything", exitCode)

	case exitPostTransfer:
		return fmt.Sprintf("pg_upgrade failed with exit code %d after conversion had begun", exitCode)
	}

	return fmt.Sprintf("pg_upgrade failed with exit code %d", exitCode)
}

// classifyOutcome maps what the clone's data directory shows to the recovery the engine has to
// perform. It deliberately ignores the script's exit code, which can be missing or misleading and
// is not needed: a present pg_control means the old cluster is startable no matter how the run
// ended, and that is also true after an engine restart, when no exit code exists at all.
func classifyOutcome(facts dataDirFacts) recoveryAction {
	// Only two things prove there is a finished cluster to promote.
	//
	// A completed swap is self-evident: swapDataDirs runs only after the conversion has already
	// been accepted, so a data directory holding the target major was put there deliberately.
	//
	// Short of that the conversion must have recorded that it finished. Neither of the other facts
	// can stand in for it: the script initdb's the new cluster before pg_upgrade is even asked to
	// check the pair, and pg_upgrade disables the old cluster before it starts moving files. A run
	// killed anywhere inside the transfer therefore presents both - and promoting on that evidence
	// would hand the user a partially linked cluster, report success, and then delete the old
	// directory holding the only copy of everything not yet linked.
	if facts.SwapCompleted || (facts.NewClusterReady && facts.OldClusterDisabled && facts.ConversionDone) {
		return recoveryComplete
	}

	// The old cluster cannot be started again and there is no finished cluster to promote, so the
	// clone has to be rebuilt from its origin snapshot. This is the path an interrupted transfer
	// takes.
	if facts.OldClusterDisabled {
		return recoveryReprovision
	}

	// The old cluster is verifiably intact, so it can be restarted whatever the exit code was.
	// This is what keeps an unrecognised failure from costing the user their data.
	return recoveryRollback
}

// validateUpgradeTarget rejects targets the upgrade image cannot serve.
func validateUpgradeTarget(currentVersion float64, targetVersion int) error {
	current := int(currentVersion)

	if current < minSourceMajor {
		return fmt.Errorf("cannot upgrade from PostgreSQL %s: the upgrade image carries binaries for %d and newer",
			formatPGVersion(currentVersion), minSourceMajor)
	}

	if targetVersion <= current {
		return fmt.Errorf("target version %d must be greater than the current version %d", targetVersion, current)
	}

	if targetVersion-current > maxMajorJump {
		return fmt.Errorf("cannot upgrade from %d to %d: the upgrade image carries binaries for at most %d preceding majors",
			current, targetVersion, maxMajorJump)
	}

	return nil
}

// upgradeContainerEnv builds the environment of the upgrade script. The collation values
// deliberately avoid the names LC_COLLATE and LC_CTYPE: those are real POSIX locale variables
// and would change the process locale of initdb and postgres instead of being carried as data.
func upgradeContainerEnv(cloneDir, dataSubDir string, state UpgradeState, req UpgradeRequest) []string {
	env := []string{
		"CLONE_DIR=" + cloneDir,
		"UPGRADE_DIR=" + upgradeDirPath(cloneDir),
		"DATA_SUBDIR=" + dataSubDir,
		"OLD_VERSION=" + state.OldVersion,
		"NEW_VERSION=" + state.NewVersion,
		"ENCODING=" + req.Encoding,
		"LC_COLLATE_VALUE=" + req.LCCollate,
		"LC_CTYPE_VALUE=" + req.LCCtype,
		"DATA_CHECKSUMS=" + req.DataChecksums,
	}

	// Left out entirely when empty: the script only passes them to initdb and pg_upgrade if it
	// receives them, and an empty --locale-provider would be rejected.
	optional := []struct {
		name  string
		value string
	}{
		{"LOCALE_PROVIDER", req.LocaleProvider},
		{"ICU_LOCALE", req.ICULocale},
		{"OLD_SERVER_OPTIONS", req.OldServerOptions},
	}

	for _, pair := range optional {
		if pair.value != "" {
			env = append(env, pair.name+"="+pair.value)
		}
	}

	return env
}

// dataDirOwner reports the numeric uid:gid owning the clone data directory. The upgrade
// container runs as that user: pg_upgrade refuses to run as root and has to write every file it
// hard-links, and the owner depends on the deployment rather than on a fixed account.
func dataDirOwner(r runners.Runner, dataDir string) (string, error) {
	out, err := r.Run("stat -c '%u:%g' "+runners.Quote(dataDir), false)
	if err != nil {
		return "", fmt.Errorf("failed to detect the owner of %s: %w", dataDir, err)
	}

	owner := strings.TrimSpace(out)
	if owner == "" || strings.HasPrefix(owner, "0:") {
		return "", fmt.Errorf("clone data directory %s is owned by %q; pg_upgrade cannot run as root", dataDir, owner)
	}

	return owner, nil
}

func upgradeContainerName(cloneName string) string {
	return "dblab_upgrade_" + cloneName
}

func upgradeDirPath(cloneDir string) string {
	return path.Join(cloneDir, upgradeDirName)
}

func newDataDir(appConfig *resources.AppConfig) string {
	return appConfig.DataDir() + "_new"
}

func backupDataDir(appConfig *resources.AppConfig) string {
	return appConfig.DataDir() + "_old"
}

func upgradeLogPath(cloneDir string) string {
	return path.Join(upgradeDirPath(cloneDir), upgradeLogsDirName, upgradeLogFileName)
}

func formatPGVersion(version float64) string {
	return strconv.FormatFloat(version, 'g', -1, 64)
}

func writeUpgradeState(cloneDir string, state UpgradeState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to encode the upgrade state: %w", err)
	}

	if err := os.WriteFile(path.Join(upgradeDirPath(cloneDir), upgradeStateFileName), data, 0644); err != nil {
		return fmt.Errorf("failed to write the upgrade state: %w", err)
	}

	return nil
}

func readUpgradeState(cloneDir string) (UpgradeState, error) {
	state := UpgradeState{}

	data, err := os.ReadFile(path.Join(upgradeDirPath(cloneDir), upgradeStateFileName))
	if err != nil {
		return state, err
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return state, fmt.Errorf("failed to decode the upgrade state: %w", err)
	}

	return state, nil
}

func writeUpgradeStage(cloneDir string, stage upgradeStage) error {
	if err := os.WriteFile(path.Join(upgradeDirPath(cloneDir), upgradeStageFileName), []byte(stage), 0644); err != nil {
		return fmt.Errorf("failed to write the upgrade stage: %w", err)
	}

	return nil
}

// readUpgradeStage never fails; the stage is only ever reported, never acted on.
func readUpgradeStage(cloneDir string) upgradeStage {
	data, err := os.ReadFile(path.Join(upgradeDirPath(cloneDir), upgradeStageFileName))
	if err != nil {
		return stageUnknown
	}

	stage := upgradeStage(strings.TrimSpace(string(data)))
	if stage == "" {
		return stageUnknown
	}

	return stage
}

// clearUpgradeArtifacts removes the state and stage files once an upgrade has settled. The log
// directory stays: it is the user's diagnostic for a failed or surprising upgrade.
func clearUpgradeArtifacts(cloneDir string) {
	for _, name := range []string{upgradeStateFileName, upgradeStageFileName} {
		if err := os.Remove(path.Join(upgradeDirPath(cloneDir), name)); err != nil && !os.IsNotExist(err) {
			log.Err("failed to remove the upgrade artifact "+name+":", err)
		}
	}
}

func readLogTail(cloneDir string) string {
	data, err := os.ReadFile(upgradeLogPath(cloneDir))
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > logTailLines {
		lines = lines[len(lines)-logTailLines:]
	}

	return strings.Join(lines, "\n")
}

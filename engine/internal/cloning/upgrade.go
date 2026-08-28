/*
2026 © Postgres.ai
*/

package cloning

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/internal/webhooks"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

// emptyPreloadLibraries scopes the old server that pg_upgrade starts on its own. Old-major
// binaries come from PGDG and carry none of the extended-postgres extension libraries, so a
// source cluster that preloads one of them would fail to start. Nothing pg_upgrade does with the
// old cluster - dumping its schema - needs a preloaded library.
const emptyPreloadLibraries = "-c shared_preload_libraries=''"

// preflightTimeout bounds the queries against the clone. They run in the background goroutine,
// but a wedged clone must not hold the upgrade in UPGRADING forever.
const preflightTimeout = 30 * time.Second

// template0LocaleQuery reads what the new cluster's initdb has to reproduce. to_jsonb keeps the
// query working across majors: the locale provider columns appeared in 15 and daticulocale was
// renamed to datlocale in 17, so they are read by name from the row instead of being selected.
const template0LocaleQuery = `
SELECT pg_encoding_to_char(d.encoding), d.datcollate, d.datctype, to_jsonb(d)
FROM pg_database d
WHERE d.datname = 'template0'`

// nonDefaultTablespacesQuery finds tablespaces that live outside the clone dataset.
const nonDefaultTablespacesQuery = `
SELECT coalesce(array_agg(spcname ORDER BY spcname), '{}')
FROM pg_tablespace
WHERE spcname NOT IN ('pg_default', 'pg_global')`

// UpgradeRequest is the resolved input of a clone major upgrade. The HTTP layer validates the
// target version and resolves the image before anything touches the clone.
type UpgradeRequest struct {
	TargetVersion int
	TargetImage   string
}

// upgradePreflight is what the new cluster's initdb has to match. It is collected while the
// clone is still running, because pg_upgrade rejects any cluster pair whose encoding or
// collation differs and the source is unreadable once its container is gone.
type upgradePreflight struct {
	Encoding       string
	LCCollate      string
	LCCtype        string
	LocaleProvider string
	ICULocale      string
	DataChecksums  string
}

// UpgradeClone starts an in-place major upgrade of a clone. It returns once the clone is marked
// UPGRADING; the outcome is observed through the clone status, like reset.
func (c *Base) UpgradeClone(cloneID string, req UpgradeRequest) error {
	w, ok := c.findWrapper(cloneID)
	if !ok {
		return models.New(models.ErrCodeNotFound, "the clone not found")
	}

	if w.Session == nil || w.Clone == nil {
		return models.New(models.ErrCodeNotFound, "clone is not started yet")
	}

	// The origin snapshot is the only way back from a failed upgrade, and the provisioner
	// dereferences it, so a clone without one must never enter the flow.
	if w.Clone.Snapshot == nil {
		return models.New(models.ErrCodeBadRequest, "clone has no origin snapshot to fall back on")
	}

	// Claiming the clone and checking its state have to happen together: two upgrades that both
	// observed OK would stop the same clone twice and race over one working directory.
	if err := c.markUpgrading(cloneID); err != nil {
		return err
	}

	// Persisted immediately, because from here on an engine restart has to be able to tell that
	// this clone was mid-upgrade. filterRunningClones deletes clones whose container is gone,
	// and the container is about to be removed.
	c.SaveClonesState()

	go c.runUpgrade(w, req)

	return nil
}

// markUpgrading claims a clone for an upgrade, refusing one that is not idle. WARNING counts as
// idle: every upgrade that did not convert anything leaves the clone running and in that status,
// and refusing it would make a single aborted attempt - a preflight timeout, a rejected target, a
// failed image pull - permanent.
func (c *Base) markUpgrading(cloneID string) error {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	w, ok := c.clones[cloneID]
	if !ok {
		return models.New(models.ErrCodeNotFound, "the clone not found")
	}

	if w.Clone.Status.Code != models.StatusOK && w.Clone.Status.Code != models.StatusWarning {
		return models.New(models.ErrCodeBadRequest,
			fmt.Sprintf("clone must have the %s or %s status to be upgraded, but it is %s",
				models.StatusOK, models.StatusWarning, w.Clone.Status.Code))
	}

	w.Clone.Status = models.Status{Code: models.StatusUpgrading, Message: models.CloneMessageUpgrading}

	return nil
}

// runUpgrade collects the pre-flight data and hands the clone to the provisioner.
func (c *Base) runUpgrade(w *CloneWrapper, req UpgradeRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), preflightTimeout)
	defer cancel()

	preflight, err := c.collectUpgradePreflight(ctx, w.Clone.ID)
	if err != nil {
		// Nothing has been touched, so the clone is still running on its original version.
		c.applyUpgradeResult(w, provision.UpgradeResult{
			Outcome: provision.UpgradeRolledBack,
			Version: w.Clone.DBVersion,
			Cause:   err.Error(),
		}, nil)

		return
	}

	result, err := c.provision.UpgradeSession(w.Session, w.Clone, provision.UpgradeRequest{
		TargetVersion:    req.TargetVersion,
		TargetImage:      req.TargetImage,
		Encoding:         preflight.Encoding,
		LCCollate:        preflight.LCCollate,
		LCCtype:          preflight.LCCtype,
		LocaleProvider:   preflight.LocaleProvider,
		ICULocale:        preflight.ICULocale,
		DataChecksums:    preflight.DataChecksums,
		OldServerOptions: emptyPreloadLibraries,
	})

	c.applyUpgradeResult(w, result, err)
}

// HasUnsettledUpgrade reports whether a clone's data directory is mid-upgrade, whatever status the
// clone currently carries. The status alone is not enough for callers that must not touch a
// half-converted directory: an upgrade that fails after the clone has been stopped settles the
// clone into WARNING with data_new still present, global/pg_control still renamed away and
// upgrade/state.json still on disk, and it stays that way until the next engine start recovers it.
// This is the same fact clonesToRecover selects on, which is why it is the right one to refuse a
// snapshot on.
func (c *Base) HasUnsettledUpgrade(cloneID string) bool {
	// The status and the fields the clone directory is derived from are taken in one critical
	// section: markUpgrading writes the status under the write lock, and reprovisionAfterUpgrade
	// and ResetClone bump Revision under it. Reading them separately could stat the directory of
	// a revision that no longer exists and let a snapshot through.
	c.cloneMutex.RLock()

	w, ok := c.clones[cloneID]
	if !ok || w.Clone == nil || w.Session == nil {
		c.cloneMutex.RUnlock()

		return false
	}

	upgrading := w.Clone.Status.Code == models.StatusUpgrading
	clone, session := *w.Clone, *w.Session

	c.cloneMutex.RUnlock()

	if upgrading {
		return true
	}

	// Deliberately outside the lock: this stats the clone directory, and holding a cloning lock
	// across disk I/O is a pattern worth not adding a second instance of.
	return c.provision.HasPendingUpgrade(&session, &clone)
}

// RecoverInterruptedUpgrades settles clones that were still upgrading when the engine stopped.
//
// It has to run before filterRunningClones: an interrupted upgrade has no container, the filter
// drops container-less clones from the registry, and cleanupInvalidClones then destroys their
// datasets. Recovering first means the filters see ordinary running clones and need no special
// case. Rebuilding a clone without asking is the right default here precisely because nobody is
// watching - the alternative is silent destruction.
func (c *Base) RecoverInterruptedUpgrades() {
	for _, w := range c.clonesToRecover() {
		log.Msg("Recovering an upgrade interrupted by an engine restart, clone:", w.Clone.ID)

		result, err := c.provision.RecoverUpgrade(w.Session, w.Clone)
		c.applyUpgradeResult(w, result, err)
	}
}

// clonesToRecover lists the clones whose upgrade has not settled. The status alone cannot decide
// that: an upgrade that fails after the clone has been stopped - a converted data directory whose
// clone then fails to start, say - records a status of its own, and a clone skipped here is a clone
// the restore path deletes and the cleanup pass destroys the dataset of. The upgrade state on disk
// is the same fact RecoverUpgrade acts on and outlives any status, so it decides here as well.
func (c *Base) clonesToRecover() []*CloneWrapper {
	c.cloneMutex.RLock()
	defer c.cloneMutex.RUnlock()

	recovering := make([]*CloneWrapper, 0)

	for _, w := range c.clones {
		if w.Clone == nil || w.Session == nil {
			continue
		}

		if w.Clone.Status.Code == models.StatusUpgrading || c.provision.HasPendingUpgrade(w.Session, w.Clone) {
			recovering = append(recovering, w)
		}
	}

	return recovering
}

// collectUpgradePreflight reads the source cluster's encoding, collation and checksum settings,
// and refuses the upgrade for a layout the clone dataset cannot carry.
func (c *Base) collectUpgradePreflight(ctx context.Context, cloneID string) (upgradePreflight, error) {
	preflight := upgradePreflight{}

	conn, err := c.ConnectToClone(ctx, cloneID)
	if err != nil {
		return preflight, errors.Wrap(err, "failed to connect to the clone")
	}

	defer func() { _ = conn.Close(ctx) }()

	tablespaces := []string{}
	if err := conn.QueryRow(ctx, nonDefaultTablespacesQuery).Scan(&tablespaces); err != nil {
		return preflight, errors.Wrap(err, "failed to check tablespaces")
	}

	if len(tablespaces) > 0 {
		return preflight, models.New(models.ErrCodeBadRequest, fmt.Sprintf(
			"clone uses non-default tablespaces (%s); they live outside the clone dataset and cannot be upgraded",
			strings.Join(tablespaces, ", ")))
	}

	row := map[string]any{}
	if err := conn.QueryRow(ctx, template0LocaleQuery).Scan(
		&preflight.Encoding, &preflight.LCCollate, &preflight.LCCtype, &row); err != nil {
		return preflight, errors.Wrap(err, "failed to read the locale of the source cluster")
	}

	provider, err := localeProvider(row)
	if err != nil {
		return preflight, err
	}

	preflight.LocaleProvider = provider

	if provider == localeProviderICU {
		preflight.ICULocale = icuLocale(row)
	}

	if err := conn.QueryRow(ctx, "SHOW data_checksums").Scan(&preflight.DataChecksums); err != nil {
		return preflight, errors.Wrap(err, "failed to read the data checksum setting")
	}

	return preflight, nil
}

const (
	localeProviderLibc    = "libc"
	localeProviderICU     = "icu"
	localeProviderBuiltin = "builtin"
)

// localeProvider maps pg_database.datlocprovider onto the initdb --locale-provider value. The
// column only exists since PostgreSQL 15; an older source yields an empty provider and initdb is
// left on its default.
func localeProvider(row map[string]any) (string, error) {
	code, _ := row["datlocprovider"].(string)

	switch code {
	case "":
		return "", nil

	case "c":
		return localeProviderLibc, nil

	case "i":
		return localeProviderICU, nil

	case "b":
		// The builtin provider takes --builtin-locale rather than --icu-locale, so accepting it
		// here would silently initdb a cluster pg_upgrade then rejects.
		return "", models.New(models.ErrCodeBadRequest,
			"clone uses the "+localeProviderBuiltin+" locale provider, which the upgrade image does not support yet")
	}

	return "", models.New(models.ErrCodeBadRequest, fmt.Sprintf("unknown locale provider %q in the source cluster", code))
}

// icuLocale reads the ICU locale, which pg_database exposes as daticulocale in 15 and 16 and as
// datlocale from 17 on.
func icuLocale(row map[string]any) string {
	if locale, ok := row["daticulocale"].(string); ok && locale != "" {
		return locale
	}

	locale, _ := row["datlocale"].(string)

	return locale
}

// applyUpgradeResult records the outcome on the clone and notifies the outside world.
//
// FATAL is reserved for a clone that is not running: filterRunningClones deletes FATAL wrappers
// and cleanupInvalidClones destroys their datasets, so using it for a clone that is alive would
// hand a working clone to the next restart's garbage collector.
func (c *Base) applyUpgradeResult(w *CloneWrapper, result provision.UpgradeResult, upgradeErr error) {
	cloneID := w.Clone.ID

	if upgradeErr != nil {
		log.Errf("failed to upgrade clone %s: %v", cloneID, upgradeErr)
		c.finishUpgrade(cloneID, upgradeFailureStatus(result, upgradeErr))

		return
	}

	if result.Outcome == provision.UpgradeRequiresReprovision {
		result = c.reprovisionAfterUpgrade(w, result)
	}

	c.applyUpgradeVersion(w, result)
	c.finishUpgrade(cloneID, upgradeStatus(result))

	if result.Outcome != provision.UpgradeSucceeded {
		log.Msg(fmt.Sprintf("Clone %s was not upgraded (%s): %s", cloneID, result.Outcome, result.Cause))
		return
	}

	c.webhookCh <- webhooks.CloneEvent{
		BasicEvent: webhooks.BasicEvent{
			EventType: webhooks.CloneUpgradeEvent,
			EntityID:  cloneID,
		},
		Host:          c.config.AccessHost,
		Port:          w.Session.Port,
		Username:      w.Clone.DB.Username,
		DBName:        w.Clone.DB.DBName,
		ContainerName: cloneID,
	}

	c.tm.SendEvent(context.Background(), telemetry.CloneUpgradeEvent, telemetry.CloneUpgraded{
		ID:              util.HashID(cloneID),
		PreviousVersion: result.PreviousVersion,
		NewVersion:      result.Version,
	})
}

func (c *Base) finishUpgrade(cloneID string, status models.Status) {
	if err := c.UpdateCloneStatus(cloneID, status); err != nil {
		log.Errf("failed to update clone status: %v", err)
	}

	c.SaveClonesState()
}

// reprovisionAfterUpgrade rebuilds a clone whose old cluster pg_upgrade already disabled. This
// lives here rather than in the provisioner because re-provisioning has to bump the clone
// revision when the clone has dependent snapshots, exactly as ResetClone does - destroying a
// dataset that has children fails otherwise.
func (c *Base) reprovisionAfterUpgrade(w *CloneWrapper, result provision.UpgradeResult) provision.UpgradeResult {
	if c.hasDependentSnapshots(w) {
		log.Warn("clone has dependent snapshots", w.Clone.ID)

		c.cloneMutex.Lock()
		w.Clone.Revision++
		c.cloneMutex.Unlock()
	}

	snapshot, err := c.provision.ResetSession(w.Session, w.Clone, result.SnapshotID)
	if err != nil {
		log.Errf("failed to restore clone %s from snapshot %s: %v", w.Clone.ID, result.SnapshotID, err)

		result.Outcome = ""
		result.Cause = fmt.Sprintf("%s; the clone could not be restored from snapshot %s either: %s",
			result.Cause, result.SnapshotID, err)

		return result
	}

	c.cloneMutex.Lock()
	w.Clone.Snapshot = snapshot
	c.cloneMutex.Unlock()

	result.Outcome = provision.UpgradeReprovisioned

	return result
}

// applyUpgradeVersion keeps the clone's recorded image and version in step with what it actually
// runs. A rollback changes neither, and a re-provisioned clone is back on the engine default, so
// its override is cleared rather than pinned to the default image.
func (c *Base) applyUpgradeVersion(w *CloneWrapper, result provision.UpgradeResult) {
	c.cloneMutex.Lock()
	defer c.cloneMutex.Unlock()

	switch result.Outcome {
	case provision.UpgradeSucceeded:
		w.Clone.DockerImage = result.Image
		w.Clone.DBVersion = result.Version

	case provision.UpgradeReprovisioned:
		w.Clone.DockerImage = ""
		w.Clone.DBVersion = ""

	case provision.UpgradeUnchanged:
		// The data directory is the only source that cannot disagree with what the clone runs.
		if result.Version != "" {
			w.Clone.DBVersion = result.Version
		}
	}
}

// upgradeFailureStatus picks the status of an upgrade that left the clone without a container.
// FATAL costs the clone its dataset - filterRunningClones drops FATAL wrappers and
// cleanupInvalidClones destroys what they pointed at - so it is only right for a clone the engine
// can do nothing more about. While the upgrade state is still on disk the next engine start
// recovers the clone instead, which for a converted data directory means bringing it up on the new
// major rather than throwing away a finished upgrade.
func upgradeFailureStatus(result provision.UpgradeResult, upgradeErr error) models.Status {
	cause := errors.Cause(upgradeErr).Error()

	if !result.Recoverable {
		return models.Status{Code: models.StatusFatal, Message: cause}
	}

	return models.Status{
		Code: models.StatusWarning,
		Message: fmt.Sprintf("%s The clone is not running; the upgrade is finished or undone when the engine restarts. Cause: %s",
			models.CloneMessageUpgradeUnsettled, cause),
	}
}

// upgradeStatus turns an outcome into the clone status a user sees. Only a clone that is not
// running is FATAL; a clone that is up, whatever version it ended up on, is a warning at worst.
func upgradeStatus(result provision.UpgradeResult) models.Status {
	switch result.Outcome {
	case provision.UpgradeSucceeded:
		return models.Status{
			Code:    models.StatusOK,
			Message: fmt.Sprintf("Clone has been upgraded to PostgreSQL %s.", result.Version),
		}

	case provision.UpgradeUnchanged:
		return models.Status{Code: models.StatusOK, Message: models.CloneMessageOK}

	case provision.UpgradeRolledBack:
		return models.Status{
			Code: models.StatusWarning,
			Message: joinNonEmpty(models.CloneMessageUpgradeWarning, stillRunningClause(result.Version),
				"Cause: "+result.Cause, logReference(result)),
		}

	case provision.UpgradeReprovisioned:
		return models.Status{
			Code: models.StatusWarning,
			Message: joinNonEmpty(fmt.Sprintf(
				"%s The clone was restored from its snapshot on PostgreSQL %s, so data written since it was created is lost. Cause: %s",
				models.CloneMessageUpgradeWarning, result.Version, result.Cause), logReference(result)),
		}
	}

	// No outcome means recovery itself failed and the clone is not running.
	return models.Status{Code: models.StatusFatal, Message: result.Cause}
}

// stillRunningClause names the major a rolled-back clone is left on. Clones created before the
// upgrade feature carry no recorded version and none can be read once the attempt is over, so the
// clause states what is known instead of naming an empty version.
func stillRunningClause(version string) string {
	if version == "" {
		return "The clone is still running on its original version."
	}

	return "The clone is still running on PostgreSQL " + version + "."
}

// logReference points at the pg_upgrade log, or carries its tail when the dataset that held it
// has been re-created.
func logReference(result provision.UpgradeResult) string {
	if result.LogPath != "" {
		return "See " + result.LogPath
	}

	if result.LogTail != "" {
		return "Log tail: " + result.LogTail
	}

	return ""
}

func joinNonEmpty(parts ...string) string {
	kept := make([]string, 0, len(parts))

	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}

	return strings.Join(kept, " ")
}

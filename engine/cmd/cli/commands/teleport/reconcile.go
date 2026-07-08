/*
2026 © Postgres.ai
*/

package teleport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const hashSuffixLen = 8

const maxNameLen = 200

// CloneServiceName builds the Teleport DB resource name for a clone.
// The port suffix is always preserved; if the full name would exceed
// maxNameLen the cloneID portion is truncated in the middle.
func CloneServiceName(envID, cloneID string, port int) string {
	portSuffix := fmt.Sprintf("-%d", port)
	prefix := fmt.Sprintf("dblab-clone-%s-", envID)
	full := prefix + cloneID + portSuffix

	if len(full) <= maxNameLen {
		return full
	}

	// Truncate cloneID and append a short hash of the full name to prevent
	// collisions between different cloneIDs that share the same prefix.
	hash := shortHash(full)
	hashPart := "-" + hash
	available := maxNameLen - len(prefix) - len(hashPart) - len(portSuffix)

	if available < 1 {
		available = 1
	}

	return prefix + cloneID[:available] + hashPart + portSuffix
}

// APIServiceName builds the Teleport App resource name for the DBLab API.
func APIServiceName(envID string) string {
	name := fmt.Sprintf("dblab-api-%s", envID)
	if len(name) > maxNameLen {
		hash := shortHash(name)
		name = name[:maxNameLen-hashSuffixLen-1] + "-" + hash
	}

	return name
}

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:hashSuffixLen]
}

func newDblabClient(cfg *Config) (*dblabapi.Client, error) {
	return dblabapi.NewClient(dblabapi.Options{
		Host:              cfg.DblabURL,
		VerificationToken: cfg.DblabToken,
	})
}

func (s *service) reconcile(ctx context.Context) {
	clones, err := s.dbClient.ListClones(ctx)
	if err != nil {
		log.Errf("reconcile: failed to list clones: %v", err)
		return
	}

	registered, err := listDBs(ctx, s.cfg)
	if err != nil {
		log.Errf("reconcile: failed to list teleport dbs, skipping: %v", err)
		return
	}

	if len(clones) == 0 && len(registered) > 0 {
		log.Msg("ListClones returned empty list but registered DBs exist, skipping reconciliation to prevent mass deregistration")
		return
	}

	desired := buildDesiredDBs(clones, s.cfg.EnvironmentID)

	toAdd, toRemove := diffDBs(desired, registered)

	for name, clone := range toAdd {
		port, err := strconv.Atoi(clone.DB.Port)
		if err != nil {
			log.Errf("reconcile: invalid port %q for clone %s, skipping: %v", clone.DB.Port, name, err)
			continue
		}

		res := dbResource{Name: name, Port: port, EnvID: s.cfg.EnvironmentID, CloneID: clone.ID, OwnerUser: clone.DB.OwnerUser}

		s.mu.Lock()
		err = createDB(ctx, s.cfg, res)
		s.mu.Unlock()

		if err != nil {
			log.Errf("reconcile: failed to register db %s: %v", name, err)
		} else {
			log.Msg("reconcile: registered db", name)
		}
	}

	for name := range toRemove {
		s.mu.Lock()
		err := removeDB(ctx, s.cfg, name)
		s.mu.Unlock()

		if err != nil {
			log.Errf("reconcile: failed to remove db %s: %v", name, err)
		} else {
			log.Msg("reconcile: removed db", name)
		}
	}

	appName := APIServiceName(s.cfg.EnvironmentID)

	exists, err := appExists(ctx, s.cfg, appName)
	if err != nil {
		log.Errf("reconcile: failed to check app %s: %v", appName, err)
		return
	}

	if exists {
		return
	}

	if err := createApp(ctx, s.cfg, appName, s.cfg.DblabURL, s.cfg.EnvironmentID); err != nil {
		log.Errf("reconcile: failed to ensure api app %s: %v", appName, err)
	}
}

func buildDesiredDBs(clones []*models.Clone, envID string) map[string]*models.Clone {
	desired := make(map[string]*models.Clone, len(clones))

	for _, clone := range clones {
		port, err := strconv.Atoi(clone.DB.Port)
		if err != nil || port == 0 {
			continue
		}

		name := CloneServiceName(envID, clone.ID, port)
		desired[name] = clone
	}

	return desired
}

func diffDBs(desired map[string]*models.Clone, registered map[string]bool) (toAdd map[string]*models.Clone, toRemove map[string]bool) {
	toAdd = make(map[string]*models.Clone)
	toRemove = make(map[string]bool)

	for name, clone := range desired {
		if !registered[name] {
			toAdd[name] = clone
		}
	}

	for name := range registered {
		if _, ok := desired[name]; !ok {
			toRemove[name] = true
		}
	}

	return toAdd, toRemove
}

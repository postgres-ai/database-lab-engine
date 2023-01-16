/*
2022 © Postgres.ai
*/

package zfs

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const (
	branchProp    = "dle:branch"
	parentProp    = "dle:parent"
	childProp     = "dle:child"
	rootProp      = "dle:root"
	messageProp   = "dle:message"
	branchSep     = ","
	empty         = "-"
	defaultBranch = "main"
)

// InitBranching inits data branching.
func (m *Manager) InitBranching() error {
	branches, err := m.ListBranches()
	if err != nil {
		return err
	}

	if len(branches) > 0 {
		log.Dbg("data branching is already initialized")

		return nil
	}

	snapshots := m.SnapshotList()

	numberSnapshots := len(snapshots)

	if numberSnapshots == 0 {
		log.Dbg("no snapshots to init data branching")
		return nil
	}

	latest := snapshots[0]

	for i := numberSnapshots; i > 1; i-- {
		if err := m.SetRelation(snapshots[i-1].ID, snapshots[i-2].ID); err != nil {
			return fmt.Errorf("failed to set snapshot relations: %w", err)
		}
	}

	if err := m.AddBranchProp(defaultBranch, latest.ID); err != nil {
		return fmt.Errorf("failed to add branch property: %w", err)
	}

	log.Msg("data branching has been successfully initialized")

	return nil
}

// VerifyBranchMetadata verifies data branching metadata.
func (m *Manager) VerifyBranchMetadata() error {
	snapshots := m.SnapshotList()

	numberSnapshots := len(snapshots)

	if numberSnapshots == 0 {
		log.Dbg("no snapshots to verify data branching")
		return nil
	}

	latest := snapshots[0]

	brName, err := m.getProperty(branchProp, latest.ID)
	if err != nil {
		log.Dbg("cannot find branch for snapshot", latest.ID, err.Error())
	}

	for i := numberSnapshots; i > 1; i-- {
		if err := m.SetRelation(snapshots[i-1].ID, snapshots[i-2].ID); err != nil {
			return fmt.Errorf("failed to set snapshot relations: %w", err)
		}

		if brName == "" {
			brName, err = m.getProperty(branchProp, snapshots[i-1].ID)
			if err != nil {
				log.Dbg("cannot find branch for snapshot", snapshots[i-1].ID, err.Error())
			}
		}
	}

	if brName == "" {
		brName = defaultBranch
	}

	if err := m.AddBranchProp(brName, latest.ID); err != nil {
		return fmt.Errorf("failed to add branch property: %w", err)
	}

	log.Msg("data branching has been verified")

	return nil
}

// CreateBranch clones data as a new branch.
func (m *Manager) CreateBranch(branchName, snapshotID string) error {
	branchPath := m.config.Pool.BranchPath(branchName)

	// zfs clone -p pool@snapshot_20221019094237 pool/branch/001-branch
	cmd := []string{
		"zfs clone -p", snapshotID, branchPath,
	}

	out, err := m.runner.Run(strings.Join(cmd, " "))
	if err != nil {
		return fmt.Errorf("zfs clone error: %w. Out: %v", err, out)
	}

	return nil
}

// Snapshot takes a snapshot of the current data state.
func (m *Manager) Snapshot(snapshotName string) error {
	cmd := []string{
		"zfs snapshot -r", snapshotName,
	}

	out, err := m.runner.Run(strings.Join(cmd, " "))
	if err != nil {
		return fmt.Errorf("zfs snapshot error: %w. Out: %v", err, out)
	}

	return nil
}

// Rename renames clone.
func (m *Manager) Rename(oldName, newName string) error {
	cmd := []string{
		"zfs rename -p", oldName, newName,
	}

	out, err := m.runner.Run(strings.Join(cmd, " "))
	if err != nil {
		return fmt.Errorf("zfs renaming error: %w. Out: %v", err, out)
	}

	return nil
}

// SetMountpoint sets clone mount point.
func (m *Manager) SetMountpoint(path, name string) error {
	cmd := []string{
		"zfs set", "mountpoint=" + path, name,
	}

	out, err := m.runner.Run(strings.Join(cmd, " "))
	if err != nil {
		return fmt.Errorf("zfs mountpoint error: %w. Out: %v", err, out)
	}

	return nil
}

// ListBranches lists data pool branches.
func (m *Manager) ListBranches() (map[string]string, error) {
	cmd := fmt.Sprintf(
		`zfs list -H -t snapshot -o %s,name | grep -v "^-" | cat`, branchProp,
	)

	out, err := m.runner.Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w. Out: %v", err, out)
	}

	branches := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(out), "\n")

	const expectedColumns = 2

	for _, line := range lines {
		fields := strings.Fields(line)

		if len(fields) != expectedColumns {
			continue
		}

		if !strings.Contains(fields[0], branchSep) {
			branches[fields[0]] = fields[1]
			continue
		}

		for _, branchName := range strings.Split(fields[0], branchSep) {
			branches[branchName] = fields[1]
		}
	}

	return branches, nil
}

var repoFields = []any{"name", parentProp, childProp, branchProp, rootProp, dataStateAtLabel, messageProp}

// GetRepo provides repository details about snapshots and branches.
func (m *Manager) GetRepo() (*models.Repo, error) {
	strFields := bytes.TrimRight(bytes.Repeat([]byte(`%s,`), len(repoFields)), ",")

	cmd := fmt.Sprintf(
		`zfs list -H -t snapshot -o `+string(strFields), repoFields...,
	)

	out, err := m.runner.Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w. Out: %v", err, out)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")

	repo := models.NewRepo()

	for _, line := range lines {
		fields := strings.Fields(line)

		if len(fields) != len(repoFields) {
			log.Dbg(fmt.Sprintf("Skip invalid line: %#v\n", line))

			continue
		}

		snDetail := models.SnapshotDetails{
			ID:          fields[0],
			Parent:      fields[1],
			Child:       unwindField(fields[2]),
			Branch:      unwindField(fields[3]),
			Root:        unwindField(fields[4]),
			DataStateAt: strings.Trim(fields[5], empty),
			Message:     decodeCommitMessage(fields[6]),
		}

		repo.Snapshots[fields[0]] = snDetail

		for _, sn := range snDetail.Branch {
			if sn == "" {
				continue
			}

			repo.Branches[sn] = fields[0]
		}
	}

	return repo, nil
}

func decodeCommitMessage(field string) string {
	if field == "" || field == empty {
		return field
	}

	decodedString, err := base64.StdEncoding.DecodeString(field)
	if err != nil {
		log.Dbg(fmt.Sprintf("Unable to decode commit message: %#v\n", field))
		return field
	}

	return string(decodedString)
}

func unwindField(field string) []string {
	trimValue := strings.Trim(field, empty)

	if len(trimValue) == 0 {
		return nil
	}

	if !strings.Contains(field, branchSep) {
		return []string{trimValue}
	}

	items := make([]string, 0)
	for _, item := range strings.Split(field, branchSep) {
		items = append(items, strings.Trim(item, empty))
	}

	return items
}

// AddBranchProp adds branch to snapshot property.
func (m *Manager) AddBranchProp(branch, snapshotName string) error {
	return m.addToSet(branchProp, snapshotName, branch)
}

// DeleteBranchProp deletes branch from snapshot property.
func (m *Manager) DeleteBranchProp(branch, snapshotName string) error {
	return m.deleteFromSet(branchProp, branch, snapshotName)
}

// SetRelation sets up relation between two snapshots.
func (m *Manager) SetRelation(parent, snapshotName string) error {
	if err := m.setParent(parent, snapshotName); err != nil {
		return err
	}

	if err := m.addChild(parent, snapshotName); err != nil {
		return err
	}

	return nil
}

// DeleteChildProp deletes child from snapshot property.
func (m *Manager) DeleteChildProp(childSnapshot, snapshotName string) error {
	return m.deleteFromSet(childProp, childSnapshot, snapshotName)
}

// DeleteRootProp deletes root from snapshot property.
func (m *Manager) DeleteRootProp(branch, snapshotName string) error {
	return m.deleteFromSet(rootProp, branch, snapshotName)
}

func (m *Manager) setParent(parent, snapshotName string) error {
	return m.setProperty(parentProp, parent, snapshotName)
}

func (m *Manager) addChild(parent, snapshotName string) error {
	return m.addToSet(childProp, parent, snapshotName)
}

// SetRoot marks snapshot as a root of branch.
func (m *Manager) SetRoot(branch, snapshotName string) error {
	return m.addToSet(rootProp, snapshotName, branch)
}

// SetDSA sets value of DataStateAt to snapshot.
func (m *Manager) SetDSA(dsa, snapshotName string) error {
	return m.setProperty(dataStateAtLabel, dsa, snapshotName)
}

// SetMessage uses the given message as the commit message.
func (m *Manager) SetMessage(message, snapshotName string) error {
	encodedMessage := base64.StdEncoding.EncodeToString([]byte(message))
	return m.setProperty(messageProp, encodedMessage, snapshotName)
}

func (m *Manager) addToSet(property, snapshot, value string) error {
	original, err := m.getProperty(property, snapshot)
	if err != nil {
		return err
	}

	dirtyList := append(strings.Split(original, branchSep), value)
	uniqueList := unique(dirtyList)

	return m.setProperty(property, strings.Join(uniqueList, branchSep), snapshot)
}

// deleteFromSet deletes specific value from snapshot property.
func (m *Manager) deleteFromSet(prop, branch, snapshotName string) error {
	propertyValue, err := m.getProperty(prop, snapshotName)
	if err != nil {
		return err
	}

	originalList := strings.Split(propertyValue, branchSep)
	resultList := make([]string, 0, len(originalList)-1)

	for _, item := range originalList {
		if item != branch {
			resultList = append(resultList, item)
		}
	}

	value := strings.Join(resultList, branchSep)

	if value == "" {
		value = empty
	}

	return m.setProperty(prop, value, snapshotName)
}

func (m *Manager) getProperty(property, snapshotName string) (string, error) {
	cmd := fmt.Sprintf("zfs get -H -o value %s %s", property, snapshotName)

	out, err := m.runner.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("error when trying to get property: %w. Out: %v", err, out)
	}

	value := strings.Trim(strings.TrimSpace(out), "-")

	return value, nil
}

func (m *Manager) setProperty(property, value, snapshotName string) error {
	if value == "" {
		value = empty
	}

	cmd := fmt.Sprintf("zfs set %s=%q %s", property, value, snapshotName)

	out, err := m.runner.Run(cmd)
	if err != nil {
		return fmt.Errorf("error when trying to set property: %w. Out: %v", err, out)
	}

	return nil
}

func unique(originalList []string) []string {
	keys := make(map[string]struct{}, 0)
	branchList := make([]string, 0, len(originalList))

	for _, item := range originalList {
		if _, ok := keys[item]; !ok {
			if item == "" || item == "-" {
				continue
			}

			keys[item] = struct{}{}

			branchList = append(branchList, item)
		}
	}

	return branchList
}

// Reset rollbacks data to ZFS snapshot.
func (m *Manager) Reset(snapshotID string, _ thinclones.ResetOptions) error {
	// zfs rollback pool@snapshot_20221019094237
	cmd := fmt.Sprintf("zfs rollback %s", snapshotID)

	if out, err := m.runner.Run(cmd, true); err != nil {
		return fmt.Errorf("failed to rollback a snapshot: %w. Out: %v", err, out)
	}

	return nil
}

// DeleteBranch deletes branch.
func (m *Manager) DeleteBranch(branch string) error {
	branchName := filepath.Join(m.Pool().Name, branch)
	cmd := fmt.Sprintf("zfs destroy -R %s", branchName)

	if out, err := m.runner.Run(cmd, true); err != nil {
		return fmt.Errorf("failed to destroy branch: %w. Out: %v", err, out)
	}

	return nil
}
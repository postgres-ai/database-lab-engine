/*
2022 © Postgres.ai
*/

package zfs

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
)

const (
	branchProp  = "dle:branch"
	parentProp  = "dle:parent"
	childProp   = "dle:child"
	rootProp    = "dle:root"
	messageProp = "dle:message"
	branchSep   = ","
	empty       = "-"
)

type cmdCfg struct {
	pool string
}

// InitBranching inits data branching.
func (m *Manager) InitBranching() error {
	snapshots := m.SnapshotList()

	numberSnapshots := len(snapshots)

	if numberSnapshots == 0 {
		log.Dbg("no snapshots to init data branching")
		return nil
	}

	latest := snapshots[0]

	if getPoolPrefix(latest.ID) != m.config.Pool.Name {
		for _, s := range snapshots {
			if s.Pool == m.config.Pool.Name {
				latest = s
				break
			}
		}
	}

	latestBranchProperty, err := m.getProperty(branchProp, latest.ID)
	if err != nil {
		return fmt.Errorf("failed to read snapshot property: %w", err)
	}

	if latestBranchProperty != "" && latestBranchProperty != "-" {
		log.Dbg("data branching is already initialized")

		return nil
	}

	if err := m.AddBranchProp(branching.DefaultBranch, latest.ID); err != nil {
		return fmt.Errorf("failed to add branch property: %w", err)
	}

	leader := latest

	for i := 1; i < numberSnapshots; i++ {
		follower := snapshots[i]

		if getPoolPrefix(leader.ID) != getPoolPrefix(follower.ID) {
			continue
		}

		if err := m.SetRelation(leader.ID, follower.ID); err != nil {
			return fmt.Errorf("failed to set snapshot relations: %w", err)
		}

		brProperty, err := m.getProperty(branchProp, follower.ID)
		if err != nil {
			return fmt.Errorf("failed to read branch property: %w", err)
		}

		if brProperty == branching.DefaultBranch {
			if err := m.DeleteBranchProp(branching.DefaultBranch, follower.ID); err != nil {
				return fmt.Errorf("failed to delete default branch property: %w", err)
			}

			break
		}

		leader = follower
	}

	// If not exists pool/branch/main, init main branch dataset.
	brName := m.Pool().BranchName(m.Pool().Name, branching.DefaultBranch)

	if err := m.CreateDataset(brName); err != nil {
		return fmt.Errorf("failed to init main branch dataset: %w", err)
	}

	m.RefreshSnapshotList()

	log.Msg("data branching has been successfully initialized")

	return nil
}

func getPoolPrefix(pool string) string {
	return strings.Split(pool, "@")[0]
}

// VerifyBranchMetadata verifies data branching metadata.
func (m *Manager) VerifyBranchMetadata() error {
	snapshots := m.SnapshotList()

	numberSnapshots := len(snapshots)

	if numberSnapshots == 0 {
		log.Dbg("no snapshots to verify data branching")
		return nil
	}

	branchHeads := make(map[string]string)

	for i := numberSnapshots; i > 0; i-- {
		sn := snapshots[i-1]
		log.Dbg(sn)

		if err := m.DeleteBranchProp(sn.Branch, sn.ID); err != nil {
			return fmt.Errorf("failed to clean branch property: %w", err)
		}

		head, ok := branchHeads[sn.Branch]
		if !ok {
			branchHeads[sn.Branch] = sn.ID
			continue
		}

		if err := m.SetRelation(head, sn.ID); err != nil {
			return fmt.Errorf("failed to set snapshot relations: %w", err)
		}

		branchHeads[sn.Branch] = sn.ID
	}

	for brName, latestID := range branchHeads {
		if err := m.AddBranchProp(brName, latestID); err != nil {
			return fmt.Errorf("failed to add branch property: %w", err)
		}
	}

	log.Msg("data branching has been verified")

	return nil
}

// CreateBranch clones data as a new branch.
func (m *Manager) CreateBranch(branchName, snapshotID string) error {
	// zfs clone -p pool@snapshot_20221019094237 pool/branch/001-branch
	cmd := []string{
		"zfs clone -p", snapshotID, branchName,
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
		"zfs snapshot ", snapshotName,
	}

	out, err := m.runner.Run(strings.Join(cmd, " "))
	if err != nil {
		return fmt.Errorf("zfs snapshot error: %w. Out: %v", err, out)
	}

	return nil
}

// Move sends and receives snapshot diff.
func (m *Manager) Move(baseSnap, currentSnap, target string) error {
	cmd := fmt.Sprintf(
		"zfs send -I %s %s | zfs receive -F %s", baseSnap, currentSnap, target,
	)

	out, err := m.runner.Run(cmd)
	if err != nil {
		return fmt.Errorf("zfs moving snapshot error: %w. Out: %v", err, out)
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
	return m.listBranches()
}

// ListAllBranches lists all branches.
func (m *Manager) ListAllBranches(poolList []string) ([]models.BranchEntity, error) {
	poolFilter := ""

	if len(poolList) > 0 {
		poolFilter += "-r " + strings.Join(poolList, " ")
	}

	cmd := fmt.Sprintf(
		// Get all ZFS snapshots (-t) with options (-o) without output headers (-H).
		// Excluding snapshots without "dle:branch" property ("grep -v").
		`zfs list -H -t snapshot -o %s,name %s | grep -v "^-" | cat`, branchProp, poolFilter,
	)

	out, err := m.runner.Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w. Out: %v", err, out)
	}

	branches := make([]models.BranchEntity, 0)
	lines := strings.Split(strings.TrimSpace(out), "\n")

	const expectedColumns = 2

	for _, line := range lines {
		fields := strings.Fields(line)

		if len(fields) != expectedColumns {
			continue
		}

		dataset := branching.ParseBaseDatasetFromSnapshot(fields[1])

		if !strings.Contains(fields[0], branchSep) {
			branches = append(branches, models.BranchEntity{Name: fields[0], Dataset: dataset, SnapshotID: fields[1]})
			continue
		}

		for _, branchName := range strings.Split(fields[0], branchSep) {
			branches = append(branches, models.BranchEntity{Name: branchName, Dataset: dataset, SnapshotID: fields[1]})
		}
	}

	return branches, nil
}

func (m *Manager) listBranches() (map[string]string, error) {
	cmd := fmt.Sprintf(
		// Get ZFS snapshots (-t) with options (-o) without output headers (-H) filtered by pool (-r).
		// Excluding snapshots without "dle:branch" property ("grep -v").
		`zfs list -H -t snapshot -o %s,name -r %s | grep -v "^-" | cat`, branchProp, m.config.Pool.Name,
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

var repoFields = []any{"name", parentProp, childProp, branchProp, rootProp, dataStateAtLabel, messageProp, "clones"}

// GetRepo provides repository details about snapshots and branches filtered by data pool.
func (m *Manager) GetRepo() (*models.Repo, error) {
	return m.getRepo(cmdCfg{pool: m.config.Pool.Name})
}

// GetAllRepo provides all repository details about snapshots and branches.
func (m *Manager) GetAllRepo() (*models.Repo, error) {
	return m.getRepo(cmdCfg{})
}

func (m *Manager) getRepo(cmdCfg cmdCfg) (*models.Repo, error) {
	strFields := bytes.TrimRight(bytes.Repeat([]byte(`%s,`), len(repoFields)), ",")

	// Get ZFS snapshots (-t) with options (-o) without output headers (-H) filtered by pool (-r).
	format := `zfs list -H -t snapshot -o ` + string(strFields)
	args := repoFields

	if cmdCfg.pool != "" {
		format += " -r %s"

		args = append(args, cmdCfg.pool)
	}

	out, err := m.runner.Run(fmt.Sprintf(format, args...))
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

		dataset, _, _ := strings.Cut(fields[0], "@")

		snDetail := models.SnapshotDetails{
			ID:          fields[0],
			Parent:      fields[1],
			Child:       unwindField(fields[2]),
			Branch:      unwindField(fields[3]),
			Root:        unwindField(fields[4]),
			DataStateAt: strings.Trim(fields[5], empty),
			Message:     decodeCommitMessage(fields[6]),
			Dataset:     dataset,
			Clones:      unwindField(fields[7]),
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

// GetSnapshotProperties get custom snapshot properties.
func (m *Manager) GetSnapshotProperties(snapshotName string) (thinclones.SnapshotProperties, error) {
	strFields := bytes.TrimRight(bytes.Repeat([]byte(`%s,`), len(repoFields)), ",")

	// Get ZFS snapshot (-t) with options (-o) without output headers (-H) filtered by snapshot.
	format := `zfs list -H -t snapshot -o ` + string(strFields) + ` %s`

	args := append(repoFields, snapshotName)

	out, err := m.runner.Run(fmt.Sprintf(format, args...))
	if err != nil {
		log.Dbg(out)

		return thinclones.SnapshotProperties{}, err
	}

	fields := strings.Fields(strings.TrimSpace(out))

	if len(fields) != len(repoFields) {
		log.Dbg("Retrieved fields values:", fields)

		return thinclones.SnapshotProperties{}, errors.New("some snapshot properties could not be retrieved")
	}

	properties := thinclones.SnapshotProperties{
		Name:        strings.Trim(fields[0], empty),
		Parent:      strings.Trim(fields[1], empty),
		Child:       strings.Trim(fields[2], empty),
		Branch:      strings.Trim(fields[3], empty),
		Root:        strings.Trim(fields[4], empty),
		DataStateAt: strings.Trim(fields[5], empty),
		Message:     decodeCommitMessage(fields[6]),
		Clones:      strings.Trim(fields[7], empty),
	}

	return properties, nil
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

	return m.addChild(parent, snapshotName)
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

// HasDependentEntity gets the root property of the snapshot.
func (m *Manager) HasDependentEntity(snapshotName string) ([]string, error) {
	root, err := m.getProperty(rootProp, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("failed to check root property: %w", err)
	}

	if root != "" {
		log.Warn(fmt.Errorf("snapshot has dependent branches: %s", root))
	}

	child, err := m.getProperty(childProp, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("failed to check snapshot child property: %w", err)
	}

	if child != "" {
		log.Warn(fmt.Sprintf("snapshot %s has dependent snapshots: %s", snapshotName, child))
	}

	clones, err := m.checkDependentClones(snapshotName)
	if err != nil {
		return nil, fmt.Errorf("failed to check dependent clones: %w", err)
	}

	dependentClones := strings.Split(clones, ",")

	return dependentClones, nil
}

// KeepRelation keeps relation between adjacent snapshots.
func (m *Manager) KeepRelation(snapshotName string) error {
	child, err := m.getProperty(childProp, snapshotName)
	if err != nil {
		return fmt.Errorf("failed to check snapshot child property: %w", err)
	}

	parent, err := m.getProperty(parentProp, snapshotName)
	if err != nil {
		return fmt.Errorf("failed to check snapshot parent property: %w", err)
	}

	if parent != "" {
		if err := m.DeleteChildProp(snapshotName, parent); err != nil {
			return fmt.Errorf("failed to delete child: %w", err)
		}

		if err := m.addChild(parent, child); err != nil {
			return fmt.Errorf("failed to add child: %w", err)
		}
	}

	if child != "" {
		if err := m.setParent(parent, child); err != nil {
			return fmt.Errorf("failed to set parent: %w", err)
		}
	}

	return nil
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

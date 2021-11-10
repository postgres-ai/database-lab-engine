/*
2019 © Postgres.ai
*/

// Package zfs provides an interface to work with ZFS.
package zfs

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/thinclones"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/util"
)

const (
	headerOffset        = 1
	dataStateAtLabel    = "dblab:datastateat"
	isRoughStateAtLabel = "dblab:isroughdsa"

	// PoolMode defines the zfs filesystem name.
	PoolMode = "zfs"
)

// ListEntry defines entry of ZFS list command.
type ListEntry struct {
	Name string

	// Read-only property that identifies the amount of disk space consumed
	// by a dataset and all its descendents.
	Used uint64

	// Controls the mount point used for this file system. When the mountpoint
	// property is changed for a file system, the file system and
	// any descendents that inherit the mount point are unmounted.
	// If the new value is legacy, then they remain unmounted. Otherwise,
	// they are automatically remounted in a new location if the property
	// was previously legacy or none, or if they were mounted before
	// the property was changed. In addition, any shared file systems are
	// unshared and shared in the new location.
	MountPoint string

	// Read-only property that identifies the compression ratio achieved for
	// a dataset, expressed as a multiplier. Compression can be enabled by the
	// zfs set compression=on dataset command.
	// The value is calculated from the logical size of all files and
	// the amount of referenced physical data. It includes explicit savings
	// through the use of the compression property.
	CompressRatio float64

	// Read-only property that identifies the amount of disk space available
	// to a file system and all its children, assuming no other activity in
	// the pool. Because disk space is shared within a pool, available space
	// can be limited by various factors including physical pool size, quotas,
	// reservations, and other datasets within the pool.
	Available uint64

	// Read-only property that identifies the dataset type as filesystem
	// (file system or clone), volume, or snapshot.
	Type string

	// Read-only property for cloned file systems or volumes that identifies
	// the snapshot from which the clone was created. The origin cannot be
	// destroyed (even with the –r or –f option) as long as a clone exists.
	// Non-cloned file systems have an origin of none.
	Origin string

	// Read-only property that identifies the date and time that a dataset
	// was created.
	Creation time.Time

	// The amount of data that is accessible by this dataset, which may
	// or may not be shared with other datasets in the pool. When a snapshot
	// or clone is created, it initially references the same amount of space
	// as the  file system or snapshot it was created from, since its contents
	// are identical.
	Referenced uint64

	// The amount of space that is "logically" accessible by this dataset.
	// See the referenced property. The logical space ignores the effect
	// of the compression and copies properties, giving a quantity closer
	// to the amount of data that applications see. However, it does include
	// space consumed by metadata.
	LogicalReferenced uint64

	// The amount of space that is "logically" consumed by this dataset
	// and all its descendents. See the used property. The logical space
	// ignores the effect of the compression and copies properties, giving
	// a quantity closer to the amount of data that applications see. However,
	// it does include space consumed by metadata.
	LogicalUsed uint64

	// The amount of space consumed by snapshots of this dataset.
	// In particular, it is the amount of space that would be freed
	// if all of this dataset's snapshots were destroyed.
	// Note that this is not simply the sum of the snapshots' used properties
	// because space can be shared by multiple snapshots.
	UsedBySnapshots uint64

	// The amount of space used by children of this dataset,
	// which would be freed if all the dataset's children were destroyed.
	UsedByChildren uint64

	// DB Lab custom fields.

	// Data state timestamp.
	DataStateAt time.Time
}

type setFunc func(s string) error

type setTuple struct {
	field   string
	setFunc setFunc
}

// EmptyPoolError defines an error when storage pool has no available elements.
type EmptyPoolError struct {
	dsType dsType
	pool   string
}

// NewEmptyPoolError creates a new EmptyPoolError.
func NewEmptyPoolError(dsType dsType, pool string) *EmptyPoolError {
	return &EmptyPoolError{dsType: dsType, pool: pool}
}

// Error prints a message describing EmptyPoolError.
func (e *EmptyPoolError) Error() string {
	return fmt.Sprintf(`no available %s for pool %q`, e.dsType, e.pool)
}

// Manager describes a filesystem manager for ZFS.
type Manager struct {
	runner runners.Runner
	config Config
}

// Config defines configuration for ZFS filesystem manager.
type Config struct {
	Pool              *resources.Pool
	PreSnapshotSuffix string
	OSUsername        string
}

// NewFSManager creates a new Manager instance for ZFS.
func NewFSManager(runner runners.Runner, config Config) *Manager {
	m := Manager{
		runner: runner,
		config: config,
	}

	return &m
}

// Pool gets a storage pool.
func (m *Manager) Pool() *resources.Pool {
	return m.config.Pool
}

// CreateClone creates a new ZFS clone.
func (m *Manager) CreateClone(cloneName, snapshotID string) error {
	exists, err := m.cloneExists(cloneName)
	if err != nil {
		return errors.Wrap(err, "clone does not exist")
	}

	if exists {
		log.Msg(fmt.Sprintf("clone %q is already exists. Skip creation", cloneName))
		return nil
	}

	clonesMountDir := m.config.Pool.ClonesDir()

	cmd := "zfs clone " +
		"-o mountpoint=" + clonesMountDir + "/" + cloneName + " " +
		snapshotID + " " +
		m.config.Pool.Name + "/" + cloneName + " && " +
		"chown -R " + m.config.OSUsername + " " + clonesMountDir + "/" + cloneName

	out, err := m.runner.Run(cmd)
	if err != nil {
		return errors.Wrapf(err, "zfs clone error. Out: %v", out)
	}

	return nil
}

// DestroyClone destroys a ZFS clone.
func (m *Manager) DestroyClone(cloneName string) error {
	exists, err := m.cloneExists(cloneName)
	if err != nil {
		return errors.Wrap(err, "clone does not exist")
	}

	if !exists {
		log.Msg(fmt.Sprintf("clone %q is not exists. Skip deletion", cloneName))
		return nil
	}

	// Delete the clone and all snapshots and clones depending on it.
	// TODO(anatoly): right now, we are using this function only for
	// deleting thin clones created by users. If we are going to use
	// this function to delete clones used during the preparation
	// of baseline snapshots, we need to omit `-R`, to avoid
	// unexpected deletion of users' clones.
	cmd := fmt.Sprintf("zfs destroy -R %s/%s", m.config.Pool.Name, cloneName)

	if _, err = m.runner.Run(cmd); err != nil {
		return errors.Wrap(err, "failed to run command")
	}

	return nil
}

// cloneExists checks whether a ZFS clone exists.
func (m *Manager) cloneExists(name string) (bool, error) {
	listZfsClonesCmd := "zfs list"

	out, err := m.runner.Run(listZfsClonesCmd, false)
	if err != nil {
		return false, errors.Wrap(err, "failed to list clones")
	}

	return strings.Contains(out, name), nil
}

// ListClonesNames lists ZFS clones.
func (m *Manager) ListClonesNames() ([]string, error) {
	listZfsClonesCmd := "zfs list -o name -H"

	cmdOutput, err := m.runner.Run(listZfsClonesCmd, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list clones")
	}

	cloneNames := []string{}
	poolPrefix := m.config.Pool.Name + "/"
	clonePoolPrefix := m.config.Pool.Name + "/" + util.ClonePrefix
	lines := strings.Split(strings.TrimSpace(cmdOutput), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, clonePoolPrefix) {
			cloneNames = append(cloneNames, strings.TrimPrefix(line, poolPrefix))
		}
	}

	return util.Unique(cloneNames), nil
}

// CreateSnapshot creates a new snapshot.
func (m *Manager) CreateSnapshot(poolSuffix, dataStateAt string) (string, error) {
	poolName := m.config.Pool.Name

	if poolSuffix != "" {
		poolName += "/" + poolSuffix
	}

	originalDSA := dataStateAt

	if dataStateAt == "" {
		dataStateAt = time.Now().Format(util.DataStateAtFormat)
	}

	snapshotName := getSnapshotName(poolName, dataStateAt)

	snapshotList, err := m.listSnapshots(poolName)
	if err != nil {
		var emptyErr *EmptyPoolError
		if !errors.As(err, &emptyErr) {
			return "", fmt.Errorf("failed to get a snapshot list: %w", err)
		}
	}

	for _, entry := range snapshotList {
		if entry.Name == snapshotName {
			return "", thinclones.NewSnapshotExistsError(snapshotName)
		}
	}

	cmd := fmt.Sprintf("zfs snapshot -r %s", snapshotName)

	if _, err := m.runner.Run(cmd, true); err != nil {
		return "", errors.Wrap(err, "failed to create snapshot")
	}

	cmd = fmt.Sprintf("zfs set %s=%q %s", dataStateAtLabel, strings.TrimSuffix(dataStateAt, m.config.PreSnapshotSuffix), snapshotName)

	if _, err := m.runner.Run(cmd, true); err != nil {
		return "", errors.Wrap(err, "failed to set the dataStateAt option for snapshot")
	}

	if originalDSA == "" {
		cmd = fmt.Sprintf("zfs set %s=%q %s", isRoughStateAtLabel, "1", snapshotName)

		if _, err := m.runner.Run(cmd, true); err != nil {
			return "", errors.Wrap(err, "failed to set the rough flag of dataStateAt option for snapshot")
		}
	}

	return snapshotName, nil
}

// getSnapshotName builds a snapshot name.
func getSnapshotName(pool, dataStateAt string) string {
	return fmt.Sprintf("%s@snapshot_%s", pool, dataStateAt)
}

// RollbackSnapshot rollbacks ZFS snapshot.
func RollbackSnapshot(r runners.Runner, pool string, snapshot string) error {
	cmd := fmt.Sprintf("zfs rollback -f -r %s", snapshot)

	if _, err := r.Run(cmd, true); err != nil {
		return errors.Wrap(err, "failed to rollback a snapshot")
	}

	return nil
}

// DestroySnapshot destroys the snapshot.
func (m *Manager) DestroySnapshot(snapshotName string) error {
	cmd := fmt.Sprintf("zfs destroy -R %s", snapshotName)

	if _, err := m.runner.Run(cmd); err != nil {
		return errors.Wrap(err, "failed to run command")
	}

	return nil
}

// CleanupSnapshots destroys old snapshots considering retention limit and related clones.
func (m *Manager) CleanupSnapshots(retentionLimit int) ([]string, error) {
	clonesCmd := fmt.Sprintf("zfs list -S clones -o name,origin -H -r %s", m.config.Pool.Name)

	clonesOutput, err := m.runner.Run(clonesCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list snapshots")
	}

	busySnapshots := m.getBusySnapshotList(clonesOutput)

	cleanupCmd := fmt.Sprintf(
		"zfs list -t snapshot -H -o name -s %s -s creation -r %s | grep -v clone | head -n -%d %s"+
			"| xargs -n1 --no-run-if-empty zfs destroy -R ",
		dataStateAtLabel, m.config.Pool.Name, retentionLimit, excludeBusySnapshots(busySnapshots))

	out, err := m.runner.Run(cleanupCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to clean up snapshots")
	}

	lines := strings.Split(out, "\n")

	return lines, nil
}

func (m *Manager) getBusySnapshotList(clonesOutput string) []string {
	systemClones, userClones := make(map[string]string), make(map[string]struct{})

	userClonePrefix := m.config.Pool.Name + "/" + util.ClonePrefix

	for _, line := range strings.Split(clonesOutput, "\n") {
		cloneLine := strings.FieldsFunc(line, unicode.IsSpace)

		if len(cloneLine) != 2 || cloneLine[1] == "-" {
			continue
		}

		if strings.HasPrefix(cloneLine[0], userClonePrefix) {
			origin := cloneLine[1]

			if idx := strings.Index(origin, "@"); idx != -1 {
				origin = origin[:idx]
			}

			userClones[origin] = struct{}{}

			continue
		}

		systemClones[cloneLine[0]] = cloneLine[1]
	}

	busySnapshots := make([]string, 0, len(userClones))

	for userClone := range userClones {
		busySnapshots = append(busySnapshots, systemClones[userClone])
	}

	return busySnapshots
}

// excludeBusySnapshots excludes snapshots that match a pattern by name.
// The exclusion logic relies on the fact that snapshots have unique substrings (timestamps).
func excludeBusySnapshots(busySnapshots []string) string {
	if len(busySnapshots) == 0 {
		return ""
	}

	return fmt.Sprintf("| grep -Ev '%s' ", strings.Join(busySnapshots, "|"))
}

// GetSessionState returns a state of a session.
func (m *Manager) GetSessionState(name string) (*resources.SessionState, error) {
	entries, err := m.listFilesystems(m.config.Pool.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list filesystems")
	}

	var sEntry *ListEntry

	entryName := m.config.Pool.Name + "/" + name

	for _, entry := range entries {
		if entry.Name == entryName {
			sEntry = entry
			break
		}
	}

	if sEntry == nil {
		return nil, errors.New("cannot get session state: specified ZFS pool does not exist")
	}

	state := &resources.SessionState{
		CloneDiffSize:     sEntry.Used,
		LogicalReferenced: sEntry.LogicalReferenced,
	}

	return state, nil
}

// GetFilesystemState returns a disk state.
func (m *Manager) GetFilesystemState() (models.FileSystem, error) {
	parts := strings.SplitN(m.config.Pool.Name, "/", 2)
	if len(parts) == 0 {
		return models.FileSystem{}, errors.New("failed to get a storage pool name")
	}

	parentPool := parts[0]

	entries, err := m.listFilesystems(parentPool)
	if err != nil {
		return models.FileSystem{}, errors.Wrap(err, "failed to list filesystems")
	}

	var parentPoolEntry, poolEntry *ListEntry

	for _, entry := range entries {
		if entry.Name == parentPool {
			parentPoolEntry = entry
		}

		if entry.Name == m.config.Pool.Name {
			poolEntry = entry
		}

		if parentPoolEntry != nil && poolEntry != nil {
			break
		}
	}

	if parentPoolEntry == nil || poolEntry == nil {
		return models.FileSystem{}, errors.New("cannot get disk state: pool entries not found")
	}

	fileSystem := models.FileSystem{
		Mode:            PoolMode,
		Size:            parentPoolEntry.Available + parentPoolEntry.Used,
		Free:            parentPoolEntry.Available,
		Used:            parentPoolEntry.Used,
		UsedBySnapshots: parentPoolEntry.UsedBySnapshots,
		UsedByClones:    parentPoolEntry.UsedByChildren,
		DataSize:        poolEntry.LogicalReferenced,
		CompressRatio:   parentPoolEntry.CompressRatio,
	}

	return fileSystem, nil
}

// GetSnapshots returns a snapshot list.
func (m *Manager) GetSnapshots() ([]resources.Snapshot, error) {
	entries, err := m.listSnapshots(m.config.Pool.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	snapshots := make([]resources.Snapshot, 0, len(entries))

	for _, entry := range entries {
		// Filter pre-snapshots, they will not be allowed to be used for cloning.
		if strings.HasSuffix(entry.Name, m.config.PreSnapshotSuffix) {
			continue
		}

		snapshot := resources.Snapshot{
			ID:                entry.Name,
			CreatedAt:         entry.Creation,
			DataStateAt:       entry.DataStateAt,
			Used:              entry.Used,
			LogicalReferenced: entry.LogicalReferenced,
			Pool:              m.config.Pool.Name,
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// ListFilesystems lists ZFS file systems (clones, pools).
func (m *Manager) listFilesystems(pool string) ([]*ListEntry, error) {
	filter := snapshotFilter{
		fields:  defaultFields,
		sorting: defaultSorting,
		pool:    pool,
		dsType:  fileSystemType,
	}

	return m.listDetails(filter)
}

// ListSnapshots lists ZFS snapshots.
func (m *Manager) listSnapshots(pool string) ([]*ListEntry, error) {
	filter := snapshotFilter{
		fields:  defaultFields,
		sorting: defaultSorting,
		pool:    pool,
		dsType:  snapshotType,
	}

	listEntries, err := m.listDetails(filter)
	if err != nil {
		return nil, err
	}

	m.calculateEntrySize(listEntries)

	return listEntries, err
}
func (m *Manager) calculateEntrySize(listEntries []*ListEntry) {
	// TODO: The `go-libzfs` library might be useful to avoid a lot of command actions: https://github.com/bicomsystems/go-libzfs
	const preCloneParts = 2

	for _, entry := range listEntries {
		// Extract the pre-clone name.
		splitEntry := strings.SplitN(entry.Name, "@", preCloneParts)
		if len(splitEntry) < preCloneParts {
			continue
		}

		preClone := splitEntry[0]

		if !strings.Contains(preClone, "_pre") {
			continue
		}

		// Get the pre-clone origin.
		preSnapshot, err := m.runner.Run(buildOriginCommand(preClone), false)
		if err != nil {
			log.Err("failed to get pre-clone origin", err.Error())
			continue
		}

		// Get the pre-snapshot size.
		sizeStr, err := m.runner.Run(buildSizeCommand(preSnapshot), false)
		if err != nil {
			log.Err("failed to get pre-snapshot size:", err.Error())
			continue
		}

		usedByPreSnapshot, err := util.ParseBytes(sizeStr)
		if err != nil {
			log.Err("cannot parse the extracted size of a pre-snapshot:", err)
			continue
		}

		entry.Used += usedByPreSnapshot
	}
}

func buildOriginCommand(clone string) string {
	return "zfs get -H -o value origin " + clone
}

func buildSizeCommand(snapshot string) string {
	return "zfs get -H -p -o value used " + snapshot
}

// listDetails lists all ZFS types.
func (m *Manager) listDetails(filter snapshotFilter) ([]*ListEntry, error) {
	// TODO(anatoly): Return map.
	// TODO(anatoly): Generalize.
	out, err := m.runner.Run(buildListCommand(filter), false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list details")
	}

	lines := strings.Split(out, "\n")

	// First line is header.
	if len(lines) <= headerOffset {
		return nil, NewEmptyPoolError(filter.dsType, filter.pool)
	}

	numberFields := len([]string(filter.fields)) // 14
	entries := make([]*ListEntry, len(lines)-headerOffset)

	for i := headerOffset; i < len(lines); i++ {
		fields := strings.Fields(lines[i])

		// Empty value of standard ZFS params is "-", but for custom
		// params it will be just an empty string. Which mean that fields
		// array contain less elements. It's still bad to not have our
		// custom variables, but we don't want fail completely in this case.
		if len(fields) == numberFields-1 {
			log.Dbg(fmt.Sprintf("Probably %q is not set. Manually check ZFS snapshots.", dataStateAtLabel))

			fields = append(fields, "-")
		}

		// In other cases something really wrong with output format.
		if len(fields) != numberFields {
			return nil, errors.Errorf("ZFS error: some fields are empty. First of all, check " + dataStateAtLabel)
		}

		zfsListEntry := &ListEntry{
			Name:       fields[0],
			MountPoint: fields[2],
			Type:       fields[5],
			Origin:     fields[6],
		}

		setRules := []setTuple{
			{field: fields[1], setFunc: zfsListEntry.setUsed},
			{field: fields[3], setFunc: zfsListEntry.setCompressRatio},
			{field: fields[4], setFunc: zfsListEntry.setAvailable},
			{field: fields[7], setFunc: zfsListEntry.setCreation},
			{field: fields[8], setFunc: zfsListEntry.setReferenced},
			{field: fields[9], setFunc: zfsListEntry.setLogicalReferenced},
			{field: fields[10], setFunc: zfsListEntry.setLogicalUsed},
			{field: fields[11], setFunc: zfsListEntry.setUsedBySnapshots},
			{field: fields[12], setFunc: zfsListEntry.setUsedByChildren},
			{field: fields[13], setFunc: zfsListEntry.setDataStateAt},
		}

		for _, rule := range setRules {
			if len(rule.field) == 0 || rule.field == "-" {
				continue
			}

			if err := rule.setFunc(rule.field); err != nil {
				return nil, errors.Errorf("ZFS error: cannot parse output.\nCommand: %s.\nOutput: %s\nErr: %v",
					buildListCommand(filter), out, err)
			}
		}

		entries[i-1] = zfsListEntry
	}

	return entries, nil
}

func (z *ListEntry) setUsed(field string) error {
	used, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.Used = used

	return nil
}

func (z *ListEntry) setCompressRatio(field string) error {
	ratioStr := strings.ReplaceAll(field, "x", "")

	compressRatio, err := strconv.ParseFloat(ratioStr, 64)
	if err != nil {
		return err
	}

	z.CompressRatio = compressRatio

	return nil
}

func (z *ListEntry) setAvailable(field string) error {
	available, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.Available = available

	return nil
}

func (z *ListEntry) setCreation(field string) error {
	creation, err := util.ParseUnixTime(field)
	if err != nil {
		return err
	}

	z.Creation = creation

	return nil
}

func (z *ListEntry) setReferenced(field string) error {
	referenced, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.Referenced = referenced

	return nil
}

func (z *ListEntry) setLogicalReferenced(field string) error {
	logicalReferenced, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.LogicalReferenced = logicalReferenced

	return nil
}

func (z *ListEntry) setLogicalUsed(field string) error {
	logicalUsed, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.LogicalUsed = logicalUsed

	return nil
}

func (z *ListEntry) setUsedBySnapshots(field string) error {
	usedBySnapshots, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.UsedBySnapshots = usedBySnapshots

	return nil
}

func (z *ListEntry) setUsedByChildren(field string) error {
	usedByChildren, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.UsedByChildren = usedByChildren

	return nil
}

func (z *ListEntry) setDataStateAt(field string) error {
	stateAt, err := util.ParseCustomTime(field)
	if err != nil {
		return err
	}

	z.DataStateAt = stateAt

	return nil
}

// PoolMappings provides a mapping of pool name and mount point directory.
func PoolMappings(runner runners.Runner, mountDir, preSnapshotSuffix string) (map[string]string, error) {
	listCmd := "zfs list -Ho name,mountpoint -t filesystem | grep -v " + preSnapshotSuffix

	output, err := runner.Run(listCmd, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list pools")
	}

	return processMappingOutput(output, mountDir), nil
}

func processMappingOutput(commandOutput, mountDir string) map[string]string {
	poolMappings := make(map[string]string)

	lines := strings.Split(commandOutput, "\n")

	for i := range lines {
		fields := strings.Fields(lines[i])

		const poolFieldsNum = 2

		if len(fields) < poolFieldsNum {
			log.Dbg("Mapping fields not found: ", fields)
			continue
		}

		// Select pools from the first nested level.
		trimPrefix := strings.TrimPrefix(fields[1], mountDir)
		poolDir := strings.Trim(trimPrefix, "/")
		baseDir := path.Base(fields[1])

		if poolDir == "" || baseDir != poolDir {
			continue
		}

		poolMappings[poolDir] = fields[0]
	}

	return poolMappings
}

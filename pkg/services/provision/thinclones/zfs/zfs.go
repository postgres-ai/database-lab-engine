/*
2019 © Postgres.ai
*/

// Package zfs provides an interface to work with ZFS.
package zfs

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/services/provision/runners"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

const (
	headerOffset        = 1
	dataStateAtLabel    = "dblab:datastateat"
	isRoughStateAtLabel = "dblab:isroughdsa"
	dataStateAtFormat   = "20060102150405"
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
	//as the  file system or snapshot it was created from, since its contents
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

	// DB Lab custom fields.

	// Data state timestamp.
	DataStateAt time.Time
}

type setFunc func(s string) error

type setTuple struct {
	field   string
	setFunc setFunc
}

// CreateClone creates a new ZFS clone.
func CreateClone(r runners.Runner, pool, name, snapshot, mountDir, osUsername string) error {
	exists, err := CloneExists(r, name)
	if err != nil {
		return errors.Wrap(err, "clone does not exist")
	}

	if exists {
		return nil
	}

	cmd := "zfs clone " +
		"-o mountpoint=" + mountDir + name + " " +
		snapshot + " " +
		pool + "/" + name + " && " +
		"chown -R " + osUsername + " " + mountDir + name

	out, err := r.Run(cmd)
	if err != nil {
		return errors.Wrapf(err, "zfs clone error. Out: %v", out)
	}

	return nil
}

// DestroyClone destroys a ZFS clone.
func DestroyClone(r runners.Runner, pool string, name string) error {
	exists, err := CloneExists(r, name)
	if err != nil {
		return errors.Wrap(err, "clone does not exist")
	}

	if !exists {
		return nil
	}

	// Delete the clone and all snapshots and clones depending on it.
	// TODO(anatoly): right now, we are using this function only for
	// deleting thin clones created by users. If we are going to use
	// this function to delete clones used during the preparation
	// of baseline snapshots, we need to omit `-R`, to avoid
	// unexpected deletion of users' clones.
	cmd := fmt.Sprintf("zfs destroy -R %s/%s", pool, name)

	if _, err = r.Run(cmd); err != nil {
		return errors.Wrap(err, "failed to run command")
	}

	return nil
}

// CloneExists checks whether a ZFS clone exists.
func CloneExists(r runners.Runner, name string) (bool, error) {
	listZfsClonesCmd := "zfs list"

	out, err := r.Run(listZfsClonesCmd, false)
	if err != nil {
		return false, errors.Wrap(err, "failed to list clones")
	}

	return strings.Contains(out, name), nil
}

// ListClones lists ZFS clones.
func ListClones(r runners.Runner, prefix string) ([]string, error) {
	listZfsClonesCmd := "zfs list"

	re := regexp.MustCompile(fmt.Sprintf(`(%s[0-9]+)`, prefix))

	out, err := r.Run(listZfsClonesCmd, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list clones")
	}

	return util.Unique(re.FindAllString(out, -1)), nil
}

// CreateSnapshot creates ZFS snapshot.
func CreateSnapshot(r runners.Runner, pool, dataStateAt string) (string, error) {
	originalDSA := dataStateAt

	if dataStateAt == "" {
		dataStateAt = time.Now().Format(dataStateAtFormat)
	}

	snapshotName := getSnapshotName(pool, dataStateAt)
	cmd := fmt.Sprintf("zfs snapshot -r %s", snapshotName)

	if _, err := r.Run(cmd, true); err != nil {
		return "", errors.Wrap(err, "failed to create snapshot")
	}

	cmd = fmt.Sprintf("zfs set %s=%q %s", dataStateAtLabel, dataStateAt, snapshotName)

	if _, err := r.Run(cmd, true); err != nil {
		return "", errors.Wrap(err, "failed to set the dataStateAt option for snapshot")
	}

	if originalDSA == "" {
		cmd = fmt.Sprintf("zfs set %s=%q %s", isRoughStateAtLabel, "1", snapshotName)

		if _, err := r.Run(cmd, true); err != nil {
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
func DestroySnapshot(r runners.Runner, snapshotName string) error {
	cmd := fmt.Sprintf("zfs destroy -R %s", snapshotName)

	if _, err := r.Run(cmd); err != nil {
		return errors.Wrap(err, "failed to run command")
	}

	return nil
}

// ListFilesystems lists ZFS file systems (clones, pools).
func ListFilesystems(r runners.Runner, pool string) ([]*ListEntry, error) {
	return ListDetails(r, pool, "filesystem")
}

// ListSnapshots lists ZFS snapshots.
func ListSnapshots(r runners.Runner, pool string) ([]*ListEntry, error) {
	return ListDetails(r, pool, "snapshot")
}

// ListDetails lists all ZFS types.
func ListDetails(r runners.Runner, pool string, dsType string) ([]*ListEntry, error) {
	// TODO(anatoly): Return map.
	// TODO(anatoly): Generalize.
	numberFields := 12
	listCmd := "zfs list -po name,used,mountpoint,compressratio,available,type," +
		"origin,creation,referenced,logicalreferenced,logicalused," + dataStateAtLabel + " " +
		"-S " + dataStateAtLabel + " -S creation " + // Order DESC.
		"-t " + dsType + " " +
		"-r " + pool

	out, err := r.Run(listCmd, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list details")
	}

	lines := strings.Split(out, "\n")

	// First line is header.
	if len(lines) <= headerOffset {
		return nil, errors.Errorf(`ZFS error: no available %s for pool %q`, dsType, pool)
	}

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
			{field: fields[11], setFunc: zfsListEntry.setDataStateAt},
		}

		for _, rule := range setRules {
			if len(rule.field) == 0 || rule.field == "-" {
				continue
			}

			if err := rule.setFunc(rule.field); err != nil {
				return nil, errors.Errorf("ZFS error: cannot parse output.\nCommand: %s.\nOutput: %s\nErr: %v",
					listCmd, out, err)
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

func (z *ListEntry) setDataStateAt(field string) error {
	stateAt, err := util.ParseCustomTime(field)
	if err != nil {
		return err
	}

	z.DataStateAt = stateAt

	return nil
}

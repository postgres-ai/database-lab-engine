/*
2019 © Postgres.ai
*/

package provision

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

const (
	headerOffset = 1
)

// ZfsListEntry defines entry of ZFS list command.
type ZfsListEntry struct {
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

// ZfsCreateClone creates a new ZFS clone.
func ZfsCreateClone(r Runner, pool, name, snapshot, mountDir, osUsername string) error {
	exists, err := ZfsCloneExists(r, name)
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

// ZfsDestroyClone destroys a ZFS clone.
func ZfsDestroyClone(r Runner, pool string, name string) error {
	exists, err := ZfsCloneExists(r, name)
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

// ZfsCloneExists checks whether a ZFS clone exists.
func ZfsCloneExists(r Runner, name string) (bool, error) {
	listZfsClonesCmd := "zfs list"

	out, err := r.Run(listZfsClonesCmd, false)
	if err != nil {
		return false, errors.Wrap(err, "failed to list clones")
	}

	return strings.Contains(out, name), nil
}

// ZfsListClones lists ZFS clones.
func ZfsListClones(r Runner, prefix string) ([]string, error) {
	listZfsClonesCmd := "zfs list"

	re := regexp.MustCompile(fmt.Sprintf(`(%s[0-9]+)`, prefix))

	out, err := r.Run(listZfsClonesCmd, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list clones")
	}

	return util.Unique(re.FindAllString(out, -1)), nil
}

// ZfsCreateSnapshot creates ZFS snapshot.
func ZfsCreateSnapshot(r Runner, pool string, snapshot string) error {
	cmd := fmt.Sprintf("zfs snapshot -r %s", snapshot)

	if _, err := r.Run(cmd, true); err != nil {
		return errors.Wrap(err, "failed to create a snapshot")
	}

	return nil
}

// ZfsRollbackSnapshot rollbacks ZFS snapshot.
func ZfsRollbackSnapshot(r Runner, pool string, snapshot string) error {
	cmd := fmt.Sprintf("zfs rollback -f -r %s", snapshot)

	if _, err := r.Run(cmd, true); err != nil {
		return errors.Wrap(err, "failed to rollback a snapshot")
	}

	return nil
}

// ZfsListFilesystems lists ZFS file systems (clones, pools).
func ZfsListFilesystems(r Runner, pool string) ([]*ZfsListEntry, error) {
	return ZfsListDetails(r, pool, "filesystem")
}

// ZfsListSnapshots lists ZFS snapshots.
func ZfsListSnapshots(r Runner, pool string) ([]*ZfsListEntry, error) {
	return ZfsListDetails(r, pool, "snapshot")
}

// ZfsListDetails lists all ZFS types.
func ZfsListDetails(r Runner, pool string, dsType string) ([]*ZfsListEntry, error) {
	// TODO(anatoly): Return map.
	// TODO(anatoly): Generalize.
	numberFields := 12
	listCmd := "zfs list " +
		"-po name,used,mountpoint,compressratio,available,type," +
		"origin,creation,referenced,logicalreferenced,logicalused," +
		"dblab:datastateat " +
		"-S dblab:datastateat -S creation " + // Order DESC.
		"-t " + dsType + " " +
		"-r " + pool

	out, err := r.Run(listCmd, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list details")
	}

	lines := strings.Split(out, "\n")

	// First line is header.
	if len(lines) <= headerOffset {
		return nil, errors.Errorf(`ZFS error: no "%s" filesystem`, pool)
	}

	entries := make([]*ZfsListEntry, len(lines)-headerOffset)

	for i := headerOffset; i < len(lines); i++ {
		fields := strings.Fields(lines[i])

		// Empty value of standard ZFS params is "-", but for custom
		// params it will be just an empty string. Which mean that fields
		// array contain less elements. It's still bad to not have our
		// custom variables, but we don't want fail completely in this case.
		if len(fields) == numberFields-1 {
			log.Dbg(`Probably "dblab:datastateat" is not set. Manually check ZFS snapshots.`)

			fields = append(fields, "-")
		}

		// In other cases something really wrong with output format.
		if len(fields) != numberFields {
			return nil, errors.Errorf("ZFS error: some fields are empty. First of all, check dblab:datastateat")
		}

		zfsListEntry := &ZfsListEntry{
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

func (z *ZfsListEntry) setUsed(field string) error {
	used, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.Used = used

	return nil
}

func (z *ZfsListEntry) setCompressRatio(field string) error {
	ratioStr := strings.ReplaceAll(field, "x", "")

	compressRatio, err := strconv.ParseFloat(ratioStr, 64)
	if err != nil {
		return err
	}

	z.CompressRatio = compressRatio

	return nil
}

func (z *ZfsListEntry) setAvailable(field string) error {
	available, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.Available = available

	return nil
}

func (z *ZfsListEntry) setCreation(field string) error {
	creation, err := util.ParseUnixTime(field)
	if err != nil {
		return err
	}

	z.Creation = creation

	return nil
}

func (z *ZfsListEntry) setReferenced(field string) error {
	referenced, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.Referenced = referenced

	return nil
}

func (z *ZfsListEntry) setLogicalReferenced(field string) error {
	logicalReferenced, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.LogicalReferenced = logicalReferenced

	return nil
}

func (z *ZfsListEntry) setLogicalUsed(field string) error {
	logicalUsed, err := util.ParseBytes(field)
	if err != nil {
		return err
	}

	z.LogicalUsed = logicalUsed

	return nil
}

func (z *ZfsListEntry) setDataStateAt(field string) error {
	stateAt, err := util.ParseCustomTime(field)
	if err != nil {
		return err
	}

	z.DataStateAt = stateAt

	return nil
}

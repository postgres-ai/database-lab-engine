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

	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

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

	// DB Lab custom fields.

	// Data state timestamp.
	DataStateAt time.Time
}

func ZfsCreateClone(r Runner, pool string, name string, snapshot string,
	mountDir string) error {
	exists, err := ZfsCloneExists(r, name)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	cmd := "sudo -n zfs clone " + snapshot + " " +
		pool + "/" + name + " -o mountpoint=" + mountDir + name + " && " +
		"sudo --non-interactive chown -R postgres " + mountDir + name

	out, err := r.Run(cmd)
	if err != nil {
		return fmt.Errorf("zfs clone error %v %v", err, out)
	}

	return nil
}

func ZfsDestroyClone(r Runner, pool string, name string) error {
	exists, err := ZfsCloneExists(r, name)
	if err != nil {
		return err
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
	cmd := fmt.Sprintf("sudo -n zfs destroy %s/%s -R", pool, name)

	_, err = r.Run(cmd)
	if err != nil {
		return err
	}

	return nil
}

func ZfsCloneExists(r Runner, name string) (bool, error) {
	listZfsClonesCmd := fmt.Sprintf(`sudo -n zfs list`)

	out, err := r.Run(listZfsClonesCmd, false)
	if err != nil {
		return false, err
	}

	return strings.Contains(out, name), nil
}

func ZfsListClones(r Runner, prefix string) ([]string, error) {
	listZfsClonesCmd := fmt.Sprintf(`sudo -n zfs list`)

	re := regexp.MustCompile(fmt.Sprintf(`(%s[0-9]+)`, prefix))

	out, err := r.Run(listZfsClonesCmd, false)
	if err != nil {
		return []string{}, err
	}

	return util.Unique(re.FindAllString(out, -1)), nil
}

func ZfsCreateSnapshot(r Runner, pool string, snapshot string) error {
	cmd := fmt.Sprintf("sudo -n zfs snapshot -r %s", snapshot)

	_, err := r.Run(cmd, true)
	if err != nil {
		return err
	}

	return nil
}

func ZfsRollbackSnapshot(r Runner, pool string, snapshot string) error {
	cmd := fmt.Sprintf("sudo -n zfs rollback -f -r %s", snapshot)

	_, err := r.Run(cmd, true)
	if err != nil {
		return err
	}

	return nil
}

func ZfsListFilesystems(r Runner, pool string) ([]*ZfsListEntry, error) {
	return ZfsListDetails(r, pool, "filesystem")
}

func ZfsListSnapshots(r Runner, pool string) ([]*ZfsListEntry, error) {
	return ZfsListDetails(r, pool, "snapshot")
}

// TODO(anatoly): Return map.
func ZfsListDetails(r Runner, pool string, dsType string) ([]*ZfsListEntry, error) {
	// TODO(anatoly): Generalize.
	numberFields := 9
	listCmd := "sudo -n zfs list " +
		"-po name,used,mountpoint,compressratio,available,type," +
		"origin,creation,dblab:datastateat " +
		"-S dblab:datastateat -S creation " + // Order DESC.
		"-t " + dsType + " " +
		"-r " + pool

	out, err := r.Run(listCmd, true)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out, "\n")

	// First line is header.
	if len(lines) < 2 {
		return nil, fmt.Errorf("ZFS error: no \"%s\" filesystem.", pool)
	}

	entries := make([]*ZfsListEntry, len(lines)-1)
	for i := 1; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) != numberFields {
			return nil, fmt.Errorf("ZFS error: some fields are empty. First of all, check dblab:datastateat.")
		}

		var (
			err1, err2, err3, err4, err5 error
			used, available              uint64
			creation, dataStateAt        time.Time
			compressRatio                float64
		)

		// Used.
		if fields[1] != "-" {
			used, err1 = strconv.ParseUint(fields[1], 10, 64)
		}

		// Compressratio.
		if fields[3] != "-" {
			ratioStr := strings.ReplaceAll(fields[3], "x", "")
			compressRatio, err2 = strconv.ParseFloat(ratioStr, 64)
		}

		// Available.
		if fields[4] != "-" {
			available, err3 = strconv.ParseUint(fields[4], 10, 64)
		}

		// Creation.
		if fields[7] != "-" {
			creationInt, err4 := strconv.ParseInt(fields[7], 10, 64)
			if err4 == nil {
				creation = time.Unix(creationInt, 0)
			}
		}

		// Dblab:datastateat.
		if fields[8] != "-" {
			dataStateAt, err5 = time.Parse("20060102030405", fields[8])
		}

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil ||
			err5 != nil {
			return nil, fmt.Errorf("ZFS error: cannot parse output.\n"+
				"Command: %s.\nOutput: %s.", listCmd, out)
		}

		entries[i-1] = &ZfsListEntry{
			Name:          fields[0],
			Used:          used,
			MountPoint:    fields[2],
			CompressRatio: compressRatio,
			Available:     available,
			Type:          fields[5],
			Origin:        fields[6],
			Creation:      creation,
			DataStateAt:   dataStateAt,
		}
	}

	return entries, nil
}

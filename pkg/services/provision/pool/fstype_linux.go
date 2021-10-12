// +build linux,!s390x,!arm,!386

/*
2020 © Postgres.ai
*/

// Package pool provides components to work with storage pools.
package pool

import (
	"strconv"
	"syscall"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/thinclones/lvm"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/thinclones/zfs"
)

var fsTypeToString = map[string]string{
	"ef53":     ext4,
	"2fc12fc1": zfs.PoolMode,
}

func (pm *Manager) getFSInfo(path string) (string, error) {
	fs := syscall.Statfs_t{}
	if err := syscall.Statfs(path, &fs); err != nil {
		return "", err
	}

	fsType := detectFSType(fs.Type)
	if fsType == ext4 {
		// cannot detect LVM checking the blockDeviceTypes map.
		return lvm.PoolMode, nil
	}

	return fsType, nil
}

// detectFSType detects the filesystem type of the underlying mounted filesystem.
func detectFSType(fsType int64) string {
	return fsTypeToString[strconv.FormatInt(fsType, 16)]
}

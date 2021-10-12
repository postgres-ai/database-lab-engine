// +build darwin freebsd dragonfly openbsd solaris

/*
2020 © Postgres.ai
*/

// Package pool provides components to work with storage pools.
package pool

import (
	"syscall"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/thinclones/lvm"
)

func (pm *Manager) getFSInfo(path string) (string, error) {
	fs := syscall.Statfs_t{}
	if err := syscall.Statfs(path, &fs); err != nil {
		return "", err
	}

	fsType := detectFSType(fs.Fstypename[:])
	if fsType == ext4 {
		// cannot detect LVM checking the blockDeviceTypes map.
		return lvm.PoolMode, nil
	}

	return fsType, nil
}

// detectFSType detects the filesystem type of the underlying mounted filesystem.
func detectFSType(fsType []int8) string {
	fsTypeBytes := make([]byte, 0, len(fsType))

	for _, v := range fsType {
		fsTypeBytes = append(fsTypeBytes, byte(v))
	}

	return string(fsTypeBytes)
}

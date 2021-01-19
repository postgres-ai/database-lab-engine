/*
2020 © Postgres.ai
*/

// Package pool provides components to work with storage pools.
package pool

import (
	"encoding/json"
	"os/exec"
	"strconv"

	"github.com/pkg/errors"
)

var fsTypeToString = map[string]string{
	"ef53":     ext4,
	"2fc12fc1": ZFS,
}

type blockDeviceList struct {
	BlockDevices []blockDevice `json:"blockdevices"`
}

type blockDevice struct {
	Type       string `json:"type"`
	MountPoint string `json:"mountpoint"`
}

// detectFSType detects the filesystem type of the underlying mounted filesystem.
func detectFSType(fsType int64) string {
	return fsTypeToString[strconv.FormatInt(fsType, 16)]
}

// getBlockDeviceTypes returns a filesystem type list of mounted block devices.
func getBlockDeviceTypes() (map[string]string, error) {
	output, err := exec.Command("lsblk", "--json", "--output", "type,mountpoint").Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run command")
	}

	var blockDevices blockDeviceList

	if err := json.Unmarshal(output, &blockDevices); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	blockDeviceTypes := make(map[string]string)

	for _, blk := range blockDevices.BlockDevices {
		if blk.MountPoint == "" || blk.Type != LVM {
			continue
		}

		blockDeviceTypes[blk.MountPoint] = blk.Type
	}

	return blockDeviceTypes, nil
}

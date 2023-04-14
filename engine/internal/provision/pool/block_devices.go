/*
2020 © Postgres.ai
*/

// Package pool provides components to work with storage pools.
package pool

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/lvm"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

type blockDeviceList struct {
	BlockDevices []blockDevice `json:"blockdevices"`
}

type blockDevice struct {
	Type       string `json:"type"`
	MountPoint string `json:"mountpoint"`
}

// getBlockDeviceTypes returns a filesystem type list of mounted block devices.
func getBlockDeviceTypes() (map[string]string, error) {
	output, err := exec.Command("lsblk", "--json", "--output", "type,mountpoint").Output()
	if err != nil {
		log.Err(output)
		return nil, fmt.Errorf("failed to run command: %w", err)
	}

	var blockDevices blockDeviceList

	if err := json.Unmarshal(output, &blockDevices); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	blockDeviceTypes := make(map[string]string)

	for _, blk := range blockDevices.BlockDevices {
		if blk.MountPoint == "" || blk.Type != lvm.PoolMode {
			continue
		}

		blockDeviceTypes[blk.MountPoint] = blk.Type
	}

	return blockDeviceTypes, nil
}

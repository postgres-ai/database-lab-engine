/*
2020 © Postgres.ai
*/

// Package lvm provides an interface to work with LVM2.
package lvm

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/runners"
)

const (
	sizePortion = 10

	// PoolMode defines the lvm filesystem name.
	PoolMode = "lvm"
)

// LvsOutput defines "lvs" command response.
type LvsOutput struct {
	Reports []ReportEntry `json:"report"`
}

// ReportEntry defines report in "lvs" command response.
type ReportEntry struct {
	Volumes []ListEntry `json:"lv"`
}

// ListEntry defines logical volume entry in "lvs" command response.
type ListEntry struct {
	Name        string `json:"lv_name"`
	GroupName   string `json:"vg_name"`
	Attr        string `json:"lv_attr"`
	Size        string `json:"lv_size"`
	Pool        string `json:"pool_lv"`
	Origin      string `json:"origin"`
	DataPercent string `json:"data_percent"` // TODO(anatoly): Float64.
}

// CreateVolume creates LVM volume.
func CreateVolume(r runners.Runner, vg, lv, name, mountDir string) error {
	fullName := getFullName(vg, name)

	volumeCreateCmd := "lvcreate --snapshot " +
		"--extents " + strconv.Itoa(sizePortion) + "%FREE " +
		"--name " + name + " " + getFullName(vg, lv)

	_, err := r.Run(volumeCreateCmd, true)
	if err != nil {
		return errors.Wrap(err, "failed to create a volume")
	}

	fullMountDir := getFullMountDir(mountDir, name)
	mountCmd := "mkdir -p " + fullMountDir + " && " +
		"mount /dev/" + fullName + " " + fullMountDir

	_, err = r.Run(mountCmd, true)
	if err != nil {
		return errors.Wrap(err, "failed to mount a volume")
	}

	return nil
}

// RemoveVolume removes LVM volume.
func RemoveVolume(r runners.Runner, vg, _, name, mountDir string) error {
	fullName := getFullName(vg, name)

	unmountCmd := "umount " + getFullMountDir(mountDir, name)

	_, err := r.Run(unmountCmd, true)
	if err != nil {
		// Can be already unmounted.
		log.Err(errors.Wrap(err, "failed to unmount volume"))
	}

	volumeRemoveCmd := fmt.Sprintf("lvremove --yes %s", fullName)

	out, err := r.Run(volumeRemoveCmd, true)
	if err != nil {
		return errors.Wrap(err, "failed to remove volume")
	}

	log.Dbg(out)

	return nil
}

// ListVolumes lists LVM volumes.
func ListVolumes(r runners.Runner, lv string) ([]ListEntry, error) {
	listVolumesCmd := `lvs --reportformat json --units b --yes ` +
		`--select origin="` + lv + `"`

	out, err := r.Run(listVolumesCmd, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list volumes")
	}

	lvsOutput := &LvsOutput{}
	if err = json.Unmarshal([]byte(out), lvsOutput); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal json: %s", out)
	}

	if len(lvsOutput.Reports) == 0 {
		return nil, errors.Errorf(`failed to parse "lvs" output`)
	}

	return lvsOutput.Reports[0].Volumes, nil
}

func getFullName(vg, name string) string {
	return fmt.Sprintf("%s/%s", vg, name)
}

func getFullMountDir(mountDir, name string) string {
	return fmt.Sprintf("%s/%s", mountDir, name)
}

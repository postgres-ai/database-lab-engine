/*
2021 © Postgres.ai
*/

package docker

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision/resources"
)

func TestSystemVolumes(t *testing.T) {
	testCases := []struct {
		path           string
		expectedSystem bool
	}{
		{path: "", expectedSystem: false},
		{path: "/var/lib/dblab", expectedSystem: false},
		{path: "/tmp", expectedSystem: false},
		{path: "/sys/kernel/debug", expectedSystem: true},
		{path: "/proc/", expectedSystem: true},
		{path: "/lib/modules", expectedSystem: true},
	}

	for _, tc := range testCases {
		assert.Equal(t, isSystemVolume(tc.path), tc.expectedSystem)
	}
}

func TestVolumesBuilding(t *testing.T) {
	testCases := []struct {
		appConfig       *resources.AppConfig
		mountPoints     []types.MountPoint
		expectedVolumes []string
	}{
		{
			appConfig: &resources.AppConfig{
				CloneName: "dblab_clone_6000",
				Pool: resources.Pool{
					Name:         "dblab_pool",
					PoolDirName:  "dblab_pool",
					MountDir:     "/var/lib/dblab/",
					CloneSubDir:  "clones",
					DataSubDir:   "data",
					SocketSubDir: "sockets",
				},
			},
			mountPoints: []types.MountPoint{
				{Source: "/lib/modules", Destination: "/lib/modules"},
				{Source: "/proc", Destination: "/host_proc"},
				{Source: "/tmp", Destination: "/tmp"},
				{Source: "/var/run/docker.sock", Destination: "/var/run/docker.sock"},
				{Source: "/sys/kernel/debug", Destination: "/sys/kernel/debug"},
				{Source: "/var/lib/dblab", Destination: "/var/lib/dblab", Propagation: "rshared"},
				{Source: "/home/user/.dblab/server.yml", Destination: "/home/dblab/configs/config.yml"},
				{Source: "/home/user/.dblab/configs", Destination: "/home/dblab/configs"},
			},
			expectedVolumes: []string{
				"--volume /var/lib/dblab/dblab_pool/sockets/dblab_clone_6000:/var/lib/dblab/dblab_pool/sockets/dblab_clone_6000:rshared",
				"--volume /var/lib/dblab/dblab_pool/clones/dblab_clone_6000/data:/var/lib/dblab/dblab_pool/clones/dblab_clone_6000/data:rshared",
			},
		},
	}

	for _, tc := range testCases {
		volumes := buildVolumesFromMountPoints(tc.appConfig, tc.mountPoints)
		assert.Equal(t, tc.expectedVolumes, volumes)
	}
}

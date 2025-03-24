/*
2021 © Postgres.ai
*/

package docker

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
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
				Branch:    "main",
				Revision:  0,
				Pool: &resources.Pool{
					Name:         "dblab_pool",
					PoolDirName:  "dblab_pool",
					MountDir:     "/var/lib/dblab/",
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
				"--volume /var/lib/dblab/dblab_pool/branch/main/dblab_clone_6000/r0:/var/lib/dblab/dblab_pool/branch/main/dblab_clone_6000/r0:rshared",
			},
		},
	}

	for _, tc := range testCases {
		volumes := buildVolumesFromMountPoints(tc.appConfig, tc.mountPoints)
		assert.Equal(t, tc.expectedVolumes, volumes)
	}
}

func TestDefaultVolumes(t *testing.T) {
	pool := resources.NewPool("test")

	pool.MountDir = "/tmp/test"
	pool.PoolDirName = "default"
	pool.SocketSubDir = "socket"

	appConfig := &resources.AppConfig{
		Pool:     pool,
		Branch:   "main",
		Revision: 0,
	}

	unixSocketCloneDir, volumes := createDefaultVolumes(appConfig)

	assert.NotEmpty(t, unixSocketCloneDir)
	assert.Equal(t, "/tmp/test/default/socket", unixSocketCloneDir)

	assert.Equal(t, 2, len(volumes))

	assert.ElementsMatch(t, []string{
		"--volume /tmp/test/default/branch/main/r0:/tmp/test/default/branch/main/r0",
		"--volume /tmp/test/default/socket:/tmp/test/default/socket"}, volumes)
}

func TestPublishPorts(t *testing.T) {
	testCases := []struct {
		provisionHosts string
		instancePort   string
		expectedResult string
	}{
		{provisionHosts: "", instancePort: "6000", expectedResult: "--publish 6000:6000"},
		{provisionHosts: "127.0.0.1", instancePort: "6000", expectedResult: "--publish 127.0.0.1:6000:6000"},
		{provisionHosts: "127.0.0.1,172.0.0.1", instancePort: "6000", expectedResult: "--publish 127.0.0.1:6000:6000 --publish 172.0.0.1:6000:6000"},
		{provisionHosts: "[::1]", instancePort: "6000", expectedResult: "--publish [::1]:6000:6000"},
	}

	for _, tc := range testCases {
		assert.Equal(t, publishPorts(tc.provisionHosts, tc.instancePort), tc.expectedResult)
	}
}

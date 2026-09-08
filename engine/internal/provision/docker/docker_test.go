/*
2021 © Postgres.ai
*/

package docker

import (
	"testing"

	"github.com/moby/moby/api/types/container"
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
		mountPoints     []container.MountPoint
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
			mountPoints: []container.MountPoint{
				{Type: "bind", Source: "/lib/modules", Destination: "/lib/modules"},
				{Type: "bind", Source: "/proc", Destination: "/host_proc"},
				{Type: "bind", Source: "/tmp", Destination: "/tmp"},
				{Type: "bind", Source: "/var/run/docker.sock", Destination: "/var/run/docker.sock"},
				{Type: "bind", Source: "/sys/kernel/debug", Destination: "/sys/kernel/debug"},
				{Type: "bind", Source: "/var/lib/dblab", Destination: "/var/lib/dblab", Propagation: "rshared"},
				{Type: "bind", Source: "/home/user/.dblab/server.yml", Destination: "/home/dblab/configs/config.yml"},
				{Type: "bind", Source: "/home/user/.dblab/configs", Destination: "/home/dblab/configs"},
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
		{provisionHosts: "", instancePort: "5432", expectedResult: "--publish 5432:5432"},
		{provisionHosts: "127.0.0.1", instancePort: "6000", expectedResult: "--publish 127.0.0.1:6000:6000"},
		{provisionHosts: "127.0.0.1,172.0.0.1", instancePort: "6000", expectedResult: "--publish 127.0.0.1:6000:6000 --publish 172.0.0.1:6000:6000"},
		{provisionHosts: "[::1]", instancePort: "6000", expectedResult: "--publish [::1]:6000:6000"},
		{provisionHosts: "0.0.0.0", instancePort: "6001", expectedResult: "--publish 0.0.0.0:6001:6001"},
		{provisionHosts: "10.0.0.1,10.0.0.2,10.0.0.3", instancePort: "6000", expectedResult: "--publish 10.0.0.1:6000:6000 --publish 10.0.0.2:6000:6000 --publish 10.0.0.3:6000:6000"},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expectedResult, publishPorts(tc.provisionHosts, tc.instancePort))
	}
}

func TestBuildSocketMount(t *testing.T) {
	testCases := []struct {
		socketDir      string
		hostDataDir    string
		destinationDir string
		expected       string
	}{
		{socketDir: "/var/lib/dblab/pool/sockets/clone1", hostDataDir: "/host/dblab", destinationDir: "/var/lib/dblab", expected: "--volume /host/dblab/pool/sockets/clone1:/var/lib/dblab/pool/sockets/clone1:rshared"},
		{socketDir: "/data/sockets/clone2", hostDataDir: "/mnt/data", destinationDir: "/data", expected: "--volume /mnt/data/sockets/clone2:/data/sockets/clone2:rshared"},
		{socketDir: "/var/lib/dblab/sockets", hostDataDir: "/var/lib/dblab", destinationDir: "/var/lib/dblab", expected: "--volume /var/lib/dblab/sockets:/var/lib/dblab/sockets:rshared"},
	}

	for _, tc := range testCases {
		result := buildSocketMount(tc.socketDir, tc.hostDataDir, tc.destinationDir)
		assert.Equal(t, tc.expected, result)
	}
}

func TestCreateDefaultVolumes_WithCloneName(t *testing.T) {
	pool := resources.NewPool("testpool")
	pool.MountDir = "/var/lib/dblab"
	pool.PoolDirName = "testpool"
	pool.SocketSubDir = "sockets"

	appConfig := &resources.AppConfig{
		CloneName: "dblab_clone_6100",
		Branch:    "main",
		Revision:  0,
		Pool:      pool,
	}

	socketDir, volumes := createDefaultVolumes(appConfig)

	assert.Equal(t, "/var/lib/dblab/testpool/sockets/dblab_clone_6100", socketDir)
	assert.Len(t, volumes, 2)
	assert.Contains(t, volumes[1], "dblab_clone_6100")
}

func TestCreateDefaultVolumes_NonZeroRevision(t *testing.T) {
	pool := resources.NewPool("pool")
	pool.MountDir = "/mnt"
	pool.PoolDirName = "pool"
	pool.SocketSubDir = "sock"
	pool.DataSubDir = "data"

	appConfig := &resources.AppConfig{
		CloneName: "clone1",
		Branch:    "dev",
		Revision:  3,
		Pool:      pool,
	}

	socketDir, volumes := createDefaultVolumes(appConfig)

	assert.Equal(t, "/mnt/pool/sock/clone1", socketDir)
	assert.Len(t, volumes, 2)
	assert.Contains(t, volumes[0], "r3")
}

func TestVolumesBuilding_NoMountPoints(t *testing.T) {
	appConfig := &resources.AppConfig{
		CloneName: "clone1",
		Branch:    "main",
		Revision:  0,
		Pool: &resources.Pool{
			Name:         "pool",
			PoolDirName:  "pool",
			MountDir:     "/var/lib/dblab",
			DataSubDir:   "data",
			SocketSubDir: "sockets",
		},
	}

	volumes := buildVolumesFromMountPoints(appConfig, nil)
	assert.Empty(t, volumes)
}

func TestVolumesBuilding_OnlySystemMounts(t *testing.T) {
	appConfig := &resources.AppConfig{
		CloneName: "clone1",
		Branch:    "main",
		Revision:  0,
		Pool: &resources.Pool{
			Name:         "pool",
			PoolDirName:  "pool",
			MountDir:     "/var/lib/dblab",
			DataSubDir:   "data",
			SocketSubDir: "sockets",
		},
	}

	mountPoints := []container.MountPoint{
		{Type: "bind", Source: "/sys/kernel/debug", Destination: "/sys/kernel/debug"},
		{Type: "bind", Source: "/proc", Destination: "/proc"},
		{Type: "bind", Source: "/lib/modules", Destination: "/lib/modules"},
	}

	volumes := buildVolumesFromMountPoints(appConfig, mountPoints)
	assert.Empty(t, volumes)
}

func TestVolumesBuilding_NonDataMountsFiltered(t *testing.T) {
	appConfig := &resources.AppConfig{
		CloneName: "clone1",
		Branch:    "main",
		Revision:  0,
		Pool: &resources.Pool{
			Name:         "pool",
			PoolDirName:  "pool",
			MountDir:     "/var/lib/dblab",
			DataSubDir:   "data",
			SocketSubDir: "sockets",
		},
	}

	mountPoints := []container.MountPoint{
		{Type: "bind", Source: "/home/user/.dblab/config.yml", Destination: "/home/dblab/configs/config.yml"},
		{Type: "bind", Source: "/tmp/logs", Destination: "/tmp/logs"},
	}

	volumes := buildVolumesFromMountPoints(appConfig, mountPoints)
	assert.Empty(t, volumes)
}

func TestSystemVolumes_BoundaryPaths(t *testing.T) {
	testCases := []struct {
		path           string
		expectedSystem bool
	}{
		{path: "/sys", expectedSystem: true},
		{path: "/lib", expectedSystem: true},
		{path: "/proc", expectedSystem: true},
		{path: "/sys/fs/cgroup", expectedSystem: true},
		{path: "/lib/x86_64-linux-gnu", expectedSystem: true},
		{path: "/proc/1/status", expectedSystem: true},
		{path: "/var/sys", expectedSystem: false},
		{path: "/var/lib/dblab", expectedSystem: false},
		{path: "/home/user", expectedSystem: false},
		{path: "/opt/data", expectedSystem: false},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expectedSystem, isSystemVolume(tc.path), "path: %s", tc.path)
	}
}

func TestPgMajorFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		env      []string
		expected int
	}{
		{name: "declared major", env: []string{"PATH=/usr/bin", "PG_MAJOR=17"}, expected: 17},
		{name: "padded value", env: []string{"PG_MAJOR= 18 "}, expected: 18},
		{name: "a later assignment shadows an earlier one", env: []string{"PG_MAJOR=16", "PG_MAJOR=17"}, expected: 17},
		{name: "an unparsable later assignment shadows a good one", env: []string{"PG_MAJOR=17", "PG_MAJOR=17-custom"}},
		{name: "no such variable", env: []string{"PATH=/usr/bin"}},
		{name: "empty environment"},
		{name: "prefix of another variable", env: []string{"PG_MAJOR_MIN=16"}},
		{name: "not a number", env: []string{"PG_MAJOR=seventeen"}},
		{name: "zero is not a major", env: []string{"PG_MAJOR=0"}},
		{name: "negative is not a major", env: []string{"PG_MAJOR=-17"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pgMajorFromEnv(tt.env))
		})
	}
}

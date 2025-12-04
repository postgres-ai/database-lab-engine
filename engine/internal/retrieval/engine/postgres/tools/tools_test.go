/*
2020 © Postgres.ai
*/

package tools

import (
	"os"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIfDirectoryEmpty(t *testing.T) {
	dirName, err := os.MkdirTemp("", "test")
	defer os.RemoveAll(dirName)

	require.NoError(t, err)

	// Check if the directory is empty.
	isEmpty, err := IsEmptyDirectory(dirName)
	require.NoError(t, err)
	assert.True(t, isEmpty)

	// Create a new file.
	_, err = os.CreateTemp(dirName, "testFile*")
	require.NoError(t, err)

	// Check if the directory is not empty.
	isEmpty, err = IsEmptyDirectory(dirName)
	require.NoError(t, err)
	assert.False(t, isEmpty)
}

func TestGetMountsFromMountPoints(t *testing.T) {
	testCases := []struct {
		name           string
		dataDir        string
		mountPoints    []container.MountPoint
		expectedPoints []mount.Mount
	}{
		{
			name:    "simple mount without transformation",
			dataDir: "/var/lib/dblab/clones/dblab_clone_6000/data",
			mountPoints: []container.MountPoint{{
				Type:        mount.TypeBind,
				Source:      "/var/lib/pgsql/data",
				Destination: "/var/lib/postgresql/data",
			}},
			expectedPoints: []mount.Mount{{
				Type:     mount.TypeBind,
				Source:   "/var/lib/pgsql/data",
				Target:   "/var/lib/postgresql/data",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					Propagation: "",
				},
			}},
		},
		{
			name:    "mount with path transformation",
			dataDir: "/var/lib/dblab/clones/dblab_clone_6000/data",
			mountPoints: []container.MountPoint{{
				Type:        mount.TypeBind,
				Source:      "/var/lib/postgresql",
				Destination: "/var/lib/dblab",
			}},
			expectedPoints: []mount.Mount{{
				Type:     mount.TypeBind,
				Source:   "/var/lib/postgresql/clones/dblab_clone_6000/data",
				Target:   "/var/lib/dblab/clones/dblab_clone_6000/data",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					Propagation: "",
				},
			}},
		},
		{
			name:    "deduplicate identical mounts",
			dataDir: "/var/lib/dblab/data",
			mountPoints: []container.MountPoint{
				{Type: mount.TypeBind, Source: "/host/dump", Destination: "/var/lib/dblab/dump"},
				{Type: mount.TypeBind, Source: "/host/dump", Destination: "/var/lib/dblab/dump"},
			},
			expectedPoints: []mount.Mount{{
				Type:     mount.TypeBind,
				Source:   "/host/dump",
				Target:   "/var/lib/dblab/dump",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					Propagation: "",
				},
			}},
		},
		{
			name:    "deduplicate mounts with trailing slashes",
			dataDir: "/var/lib/dblab/data",
			mountPoints: []container.MountPoint{
				{Type: mount.TypeBind, Source: "/host/dump/", Destination: "/var/lib/dblab/dump"},
				{Type: mount.TypeBind, Source: "/host/dump", Destination: "/var/lib/dblab/dump/"},
			},
			expectedPoints: []mount.Mount{{
				Type:     mount.TypeBind,
				Source:   "/host/dump/",
				Target:   "/var/lib/dblab/dump",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					Propagation: "",
				},
			}},
		},
		{
			name:    "volume mount uses name instead of path",
			dataDir: "/var/lib/dblab/data",
			mountPoints: []container.MountPoint{{
				Type:        mount.TypeVolume,
				Name:        "3749a7e336f27d8c1ce2a81c7b945954f7522ecc3a4be4a3855bf64473f63a89",
				Source:      "/var/lib/docker/volumes/3749a7e336f27d8c1ce2a81c7b945954f7522ecc3a4be4a3855bf64473f63a89/_data",
				Destination: "/var/lib/docker",
				RW:          false,
			}},
			expectedPoints: []mount.Mount{{
				Type:     mount.TypeVolume,
				Source:   "3749a7e336f27d8c1ce2a81c7b945954f7522ecc3a4be4a3855bf64473f63a89",
				Target:   "/var/lib/docker",
				ReadOnly: true,
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mounts := GetMountsFromMountPoints(tc.dataDir, tc.mountPoints)
			assert.Equal(t, tc.expectedPoints, mounts)
		})
	}
}

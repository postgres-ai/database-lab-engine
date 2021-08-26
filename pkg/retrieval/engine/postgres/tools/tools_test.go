/*
2020 © Postgres.ai
*/

package tools

import (
	"os"
	"testing"

	"github.com/docker/docker/api/types"
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
		dataDir        string
		mountPoints    []types.MountPoint
		expectedPoints []mount.Mount
	}{
		{
			dataDir: "/var/lib/dblab/clones/dblab_clone_6000/data",
			mountPoints: []types.MountPoint{{
				Source:      "/var/lib/pgsql/data",
				Destination: "/var/lib/postgresql/data",
			}},
			expectedPoints: []mount.Mount{{
				Source:   "/var/lib/pgsql/data",
				Target:   "/var/lib/postgresql/data",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					Propagation: "",
				},
			}},
		},

		{
			dataDir: "/var/lib/dblab/clones/dblab_clone_6000/data",
			mountPoints: []types.MountPoint{{
				Source:      "/var/lib/postgresql",
				Destination: "/var/lib/dblab",
			}},
			expectedPoints: []mount.Mount{{
				Source:   "/var/lib/postgresql/clones/dblab_clone_6000/data",
				Target:   "/var/lib/dblab/clones/dblab_clone_6000/data",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					Propagation: "",
				},
			}},
		},
	}

	for _, tc := range testCases {
		mounts := GetMountsFromMountPoints(tc.dataDir, tc.mountPoints)
		assert.Equal(t, tc.expectedPoints, mounts)
	}
}

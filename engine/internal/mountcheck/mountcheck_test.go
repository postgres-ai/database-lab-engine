/*
2026 © Postgres.ai
*/

package mountcheck

import (
	"strings"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/stretchr/testify/assert"
)

const (
	dataRoot = "/var/lib/dblab"
	dumpDir  = "/var/lib/dblab/dblab_pool/dump"
)

func TestNestedBinds(t *testing.T) {
	testCases := []struct {
		name   string
		points []container.MountPoint
		want   []nestedBind
	}{
		{
			name: "dump directory nested inside rshared parent",
			points: []container.MountPoint{
				{Destination: dataRoot, Propagation: mount.PropagationRShared},
				{Destination: dumpDir, Propagation: mount.PropagationRPrivate},
			},
			want: []nestedBind{{parent: dataRoot, child: dumpDir, propagation: mount.PropagationRShared}},
		},
		{
			name: "parent destination with trailing slash",
			points: []container.MountPoint{
				{Destination: "/var/lib/dblab/", Propagation: mount.PropagationShared},
				{Destination: dumpDir},
			},
			want: []nestedBind{{parent: dataRoot, child: dumpDir, propagation: mount.PropagationShared}},
		},
		{
			name: "parent with slave propagation does not replay mounts onto the host",
			points: []container.MountPoint{
				{Destination: dataRoot, Propagation: mount.PropagationRSlave},
				{Destination: dumpDir},
			},
			want: []nestedBind{},
		},
		{
			name: "sibling paths sharing a prefix are not nested",
			points: []container.MountPoint{
				{Destination: dataRoot, Propagation: mount.PropagationRShared},
				{Destination: "/var/lib/dblab_dumps"},
			},
			want: []nestedBind{},
		},
		{
			name: "several nested mounts are reported sorted by child",
			points: []container.MountPoint{
				{Destination: dataRoot, Propagation: mount.PropagationRShared},
				{Destination: "/var/lib/dblab/pool/dump"},
				{Destination: "/var/lib/dblab/logs"},
			},
			want: []nestedBind{
				{parent: dataRoot, child: "/var/lib/dblab/logs", propagation: mount.PropagationRShared},
				{parent: dataRoot, child: "/var/lib/dblab/pool/dump", propagation: mount.PropagationRShared},
			},
		},
		{
			name: "a child under two shared parents is reported once against the innermost",
			points: []container.MountPoint{
				{Destination: dataRoot, Propagation: mount.PropagationRShared},
				{Destination: "/var/lib/dblab/dblab_pool", Propagation: mount.PropagationRShared},
				{Destination: dumpDir},
			},
			want: []nestedBind{
				{parent: dataRoot, child: "/var/lib/dblab/dblab_pool", propagation: mount.PropagationRShared},
				{parent: "/var/lib/dblab/dblab_pool", child: dumpDir, propagation: mount.PropagationRShared},
			},
		},
		{
			name:   "no mounts",
			points: nil,
			want:   []nestedBind{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, nestedBinds(tc.points))
		})
	}
}

func TestStackedMounts(t *testing.T) {
	testCases := []struct {
		name      string
		mountInfo string
		want      []stackedMount
	}{
		{
			name: "single mounts are not reported",
			mountInfo: `25 31 0:23 / /proc rw,nosuid,nodev,noexec,relatime shared:5 - proc proc rw
100 31 8:1 /var/lib/dblab /var/lib/dblab rw,relatime shared:1 - ext4 /dev/sda1 rw
101 100 8:1 /var/lib/dblab/dblab_pool/dump /var/lib/dblab/dblab_pool/dump rw,relatime shared:1 - ext4 /dev/sda1 rw`,
			want: []stackedMount{},
		},
		{
			name: "stacked bind of the same subtree is counted",
			mountInfo: `100 31 8:1 /var/lib/dblab /var/lib/dblab rw,relatime shared:1 - ext4 /dev/sda1 rw
101 100 8:1 /var/lib/dblab/dblab_pool/dump /var/lib/dblab/dblab_pool/dump rw,relatime shared:1 - ext4 /dev/sda1 rw
102 101 8:1 /var/lib/dblab/dblab_pool/dump /var/lib/dblab/dblab_pool/dump rw,relatime shared:1 - ext4 /dev/sda1 rw
103 102 8:1 /var/lib/dblab/dblab_pool/dump /var/lib/dblab/dblab_pool/dump rw,relatime - ext4 /dev/sda1 rw`,
			want: []stackedMount{{path: dumpDir, count: 3}},
		},
		{
			name: "a different filesystem mounted on top is a real mount, not a duplicate",
			mountInfo: `100 31 8:1 /var/lib/dblab /var/lib/dblab rw,relatime shared:1 - ext4 /dev/sda1 rw
101 100 8:17 / /var/lib/dblab/dblab_pool/dump rw,relatime shared:2 - ext4 /dev/sdb1 rw`,
			want: []stackedMount{},
		},
		{
			name: "escaped whitespace in the mount point is decoded",
			mountInfo: `101 100 8:1 /dump\040dir /mnt/dump\040dir rw - ext4 /dev/sda1 rw
102 101 8:1 /dump\040dir /mnt/dump\040dir rw - ext4 /dev/sda1 rw`,
			want: []stackedMount{{path: "/mnt/dump dir", count: 2}},
		},
		{
			name: "escaped backslash in the mount point is decoded",
			mountInfo: `101 100 8:1 /dump\134dir /mnt/dump\134dir rw - ext4 /dev/sda1 rw
102 101 8:1 /dump\134dir /mnt/dump\134dir rw - ext4 /dev/sda1 rw`,
			want: []stackedMount{{path: `/mnt/dump\dir`, count: 2}},
		},
		{
			name: "an escaped backslash does not turn the digits after it into another escape",
			mountInfo: `101 100 8:1 /dump\134040 /mnt/dump\134040 rw - ext4 /dev/sda1 rw
102 101 8:1 /dump\134040 /mnt/dump\134040 rw - ext4 /dev/sda1 rw`,
			want: []stackedMount{{path: `/mnt/dump\040`, count: 2}},
		},
		{
			name:      "malformed lines are skipped",
			mountInfo: "garbage\n1 2 3\n1 2 3 4\n",
			want:      []stackedMount{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, stackedMounts(strings.NewReader(tc.mountInfo)))
		})
	}
}

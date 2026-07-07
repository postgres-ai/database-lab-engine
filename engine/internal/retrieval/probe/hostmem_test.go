/*
2026 © Postgres.ai
*/

package probe

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestDetectHostMemoryBytes(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    uint64
	}{
		{name: "typical meminfo", content: "MemTotal:       16384000 kB\nMemFree:         8000000 kB\n", want: 16384000 * 1024},
		{name: "small system", content: "MemTotal:        2048 kB\n", want: 2048 * 1024},
		{name: "missing memtotal line", content: "MemFree: 1000 kB\n", want: 0},
		{name: "malformed value", content: "MemTotal:        notanumber kB\n", want: 0},
		{name: "unexpected unit", content: "MemTotal:        16 MB\n", want: 0},
		{name: "wrong field count", content: "MemTotal:        16384000\n", want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fstest.MapFS{"proc/meminfo": &fstest.MapFile{Data: []byte(tc.content)}}
			got := DetectHostMemoryBytes(fsys)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDetectHostMemoryBytes_MissingFile(t *testing.T) {
	got := DetectHostMemoryBytes(fstest.MapFS{})
	require.Zero(t, got)
}

func TestRecommendSharedBuffers(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	const mib = uint64(1024 * 1024)

	tests := []struct {
		name  string
		bytes uint64
		want  string
	}{
		{name: "unknown memory falls back to 1GB", bytes: 0, want: "1GB"},
		{name: "256MB host floors at 128MB", bytes: 256 * mib, want: "128MB"},
		{name: "1GB host clamps at floor", bytes: 1 * gib, want: "256MB"},
		{name: "4GB host gets 1GB", bytes: 4 * gib, want: "1GB"},
		{name: "32GB host gets 8GB", bytes: 32 * gib, want: "8GB"},
		{name: "64GB host caps at 8GB", bytes: 64 * gib, want: "8GB"},
		{name: "very small host hits floor", bytes: 100 * mib, want: "128MB"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, RecommendSharedBuffers(tc.bytes))
		})
	}
}

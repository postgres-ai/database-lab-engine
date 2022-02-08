/*
2021 © Postgres.ai
*/

package cont

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceOptions(t *testing.T) {
	testCases := []struct {
		configOptions  map[string]interface{}
		expectedConfig *container.HostConfig
	}{
		{
			configOptions: map[string]interface{}{
				"memory":             100000,
				"memory-swappiness":  50,
				"memory-reservation": 3000,
				"kernel-memory":      "5m",
				"memory-swap":        "100M",
				"shm-size":           "64m",
				"oom-kill-disable":   true,

				"cpu-period":  100000,
				"cpu-quota":   100000,
				"cpuset-cpus": "1",
				"cpu-shares":  1024,

				"blkio-weight":  150,
				"oom-score-adj": 10,
			},
			expectedConfig: &container.HostConfig{
				Resources: container.Resources{
					Memory:            100000,
					MemorySwappiness:  pointer.ToInt64(50),
					MemoryReservation: 3000,
					KernelMemory:      5242880,
					MemorySwap:        104857600,
					OomKillDisable:    pointer.ToBool(true),
					CpusetCpus:        "1",
					CPUPeriod:         100000,
					CPUQuota:          100000,
					CPUShares:         1024,
					BlkioWeight:       150,
				},
				OomScoreAdj: 10,
				ShmSize:     67108864,
			},
		},
		{
			configOptions: map[string]interface{}{
				"memory":            100000,
				"memoryswappiness":  50,
				"memoryreservation": 3000,
				"kernelmemory":      "5m",
				"memoryswap":        "100M",
				"shmsize":           "1gb",
				"oomkilldisable":    true,

				"cpuperiod":  100000,
				"cpuquota":   100000,
				"cpusetcpus": "1",
				"cpushares":  1024,

				"blkioweight": 150,
				"oomscoreadj": 10,
			},
			expectedConfig: &container.HostConfig{
				Resources: container.Resources{
					Memory:            100000,
					MemorySwappiness:  pointer.ToInt64(50),
					MemoryReservation: 3000,
					KernelMemory:      5242880,
					MemorySwap:        104857600,
					OomKillDisable:    pointer.ToBool(true),
					CpusetCpus:        "1",
					CPUPeriod:         100000,
					CPUQuota:          100000,
					CPUShares:         1024,
					BlkioWeight:       150,
				},
				OomScoreAdj: 10,
				ShmSize:     1073741824,
			},
		},
	}

	for _, tc := range testCases {
		hostConfig, err := ResourceOptions(tc.configOptions)
		require.Nil(t, err)

		assert.Equal(t, tc.expectedConfig.Resources, hostConfig.Resources)
		assert.Equal(t, tc.expectedConfig.ShmSize, hostConfig.ShmSize)
	}
}

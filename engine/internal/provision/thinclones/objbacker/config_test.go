/*
2025 © Postgres.ai
*/

package objbacker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.False(t, cfg.Enabled)
	assert.Equal(t, StorageTypeS3, cfg.StorageType)
	assert.Equal(t, "/dev/zfs_objbacker", cfg.DevicePath)
	assert.Equal(t, "/var/run/zfs_objbacker.sock", cfg.DaemonSocketPath)
	assert.Equal(t, int64(1024*1024), cfg.Performance.BlockSize)
	assert.Equal(t, int64(10*1024*1024*1024), cfg.Performance.LocalCacheSize)
	assert.Equal(t, 32, cfg.Performance.MaxConcurrentOps)
	assert.True(t, cfg.Tiering.Enabled)
	assert.Equal(t, 24*time.Hour, cfg.Tiering.HotDataThreshold)
}

func TestConfig_Validate(t *testing.T) {
	t.Run("disabled config is always valid", func(t *testing.T) {
		cfg := Config{Enabled: false}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing bucket fails", func(t *testing.T) {
		cfg := Config{Enabled: true, StorageType: StorageTypeS3, Region: "us-east-1", Performance: PerformanceConfig{LocalCachePath: "/tmp"}}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket")
	})

	t.Run("invalid storage type fails", func(t *testing.T) {
		cfg := Config{Enabled: true, StorageType: "invalid", Bucket: "test", Performance: PerformanceConfig{LocalCachePath: "/tmp"}}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "storage type")
	})

	t.Run("s3 without region fails", func(t *testing.T) {
		cfg := Config{Enabled: true, StorageType: StorageTypeS3, Bucket: "test", Performance: PerformanceConfig{LocalCachePath: "/tmp"}}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region")
	})

	t.Run("s3 with iam role passes without region", func(t *testing.T) {
		cfg := Config{
			Enabled: true, StorageType: StorageTypeS3, Bucket: "test",
			Credentials: CredentialsConfig{UseIAMRole: true},
			Performance: PerformanceConfig{LocalCachePath: "/tmp"},
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing credentials fails", func(t *testing.T) {
		cfg := Config{
			Enabled: true, StorageType: StorageTypeS3, Bucket: "test", Region: "us-east-1",
			Performance: PerformanceConfig{LocalCachePath: "/tmp"},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "credentials")
	})

	t.Run("missing cache path fails", func(t *testing.T) {
		cfg := Config{
			Enabled: true, StorageType: StorageTypeS3, Bucket: "test", Region: "us-east-1",
			Credentials: CredentialsConfig{UseIAMRole: true},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "localCachePath")
	})

	t.Run("valid s3 config passes", func(t *testing.T) {
		cfg := Config{
			Enabled: true, StorageType: StorageTypeS3, Bucket: "test", Region: "us-east-1",
			Credentials: CredentialsConfig{UseIAMRole: true},
			Performance: PerformanceConfig{LocalCachePath: "/tmp"},
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid gcs config passes", func(t *testing.T) {
		cfg := Config{
			Enabled: true, StorageType: StorageTypeGCS, Bucket: "test",
			Credentials: CredentialsConfig{CredentialsFile: "/path/to/creds.json"},
			Performance: PerformanceConfig{LocalCachePath: "/tmp"},
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid azure config passes", func(t *testing.T) {
		cfg := Config{
			Enabled: true, StorageType: StorageTypeAzure, Bucket: "test",
			Credentials: CredentialsConfig{AccessKeyID: "key"},
			Performance: PerformanceConfig{LocalCachePath: "/tmp"},
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})
}

func TestConfig_ObjectPath(t *testing.T) {
	t.Run("without prefix", func(t *testing.T) {
		cfg := Config{Bucket: "test"}
		path := cfg.ObjectPath("snapshots/snap1")
		assert.Equal(t, "snapshots/snap1", path)
	})

	t.Run("with prefix", func(t *testing.T) {
		cfg := Config{Bucket: "test", Prefix: "dblab/prod"}
		path := cfg.ObjectPath("snapshots/snap1")
		assert.Equal(t, "dblab/prod/snapshots/snap1", path)
	})
}

func TestNewManager(t *testing.T) {
	t.Run("disabled config succeeds", func(t *testing.T) {
		cfg := Config{Enabled: false}
		m, err := NewManager(cfg)
		require.NoError(t, err)
		assert.NotNil(t, m)
		assert.False(t, m.IsEnabled())
	})

	t.Run("invalid config fails", func(t *testing.T) {
		cfg := Config{Enabled: true}
		m, err := NewManager(cfg)
		assert.Error(t, err)
		assert.Nil(t, m)
	})

	t.Run("valid config succeeds", func(t *testing.T) {
		cfg := Config{
			Enabled: true, StorageType: StorageTypeS3, Bucket: "test", Region: "us-east-1",
			Credentials: CredentialsConfig{UseIAMRole: true},
			Performance: PerformanceConfig{LocalCachePath: "/tmp"},
		}
		m, err := NewManager(cfg)
		require.NoError(t, err)
		assert.NotNil(t, m)
		assert.True(t, m.IsEnabled())
	})
}

func TestManager_GenerateVDEVPath(t *testing.T) {
	cfg := Config{
		Enabled: true, StorageType: StorageTypeS3, Bucket: "test", Region: "us-east-1",
		Credentials: CredentialsConfig{UseIAMRole: true},
		Performance: PerformanceConfig{LocalCachePath: "/tmp"},
		DevicePath:  "/dev/zfs_objbacker",
	}
	m, err := NewManager(cfg)
	require.NoError(t, err)

	path := m.GenerateVDEVPath("vdev123")
	assert.Equal(t, "/dev/zfs_objbacker:vdev123", path)
}

func TestManager_BuildPoolCreateCommand(t *testing.T) {
	cfg := Config{
		Enabled: true, StorageType: StorageTypeS3, Bucket: "test", Region: "us-east-1",
		Credentials: CredentialsConfig{UseIAMRole: true},
		Performance: PerformanceConfig{LocalCachePath: "/tmp"},
		DevicePath:  "/dev/zfs_objbacker",
	}
	m, err := NewManager(cfg)
	require.NoError(t, err)

	t.Run("basic pool", func(t *testing.T) {
		opts := PoolCreateOptions{
			MountPoint:   "/mnt/test",
			Properties:   map[string]string{"ashift": "12"},
			FSProperties: map[string]string{"compression": "lz4"},
		}
		args := m.BuildPoolCreateCommand("testpool", "vdev123", opts)

		assert.Contains(t, args, "create")
		assert.Contains(t, args, "-m")
		assert.Contains(t, args, "/mnt/test")
		assert.Contains(t, args, "testpool")
		assert.Contains(t, args, "/dev/zfs_objbacker:vdev123")
	})

	t.Run("tiered pool with local vdev", func(t *testing.T) {
		m.config.Tiering.Enabled = true
		opts := PoolCreateOptions{
			MountPoint: "/mnt/test",
			LocalVDEV:  "/dev/nvme0n1",
		}
		args := m.BuildPoolCreateCommand("testpool", "vdev123", opts)

		assert.Contains(t, args, "mirror")
		assert.Contains(t, args, "/dev/nvme0n1")
	})

	t.Run("pool with cache device", func(t *testing.T) {
		m.config.Tiering.Enabled = false
		opts := PoolCreateOptions{
			MountPoint:  "/mnt/test",
			CacheDevice: "/dev/sda",
		}
		args := m.BuildPoolCreateCommand("testpool", "vdev123", opts)

		assert.Contains(t, args, "cache")
		assert.Contains(t, args, "/dev/sda")
	})
}

func TestManager_EstimateCost(t *testing.T) {
	tests := []struct {
		name        string
		storageType StorageType
		dataSize    uint64
	}{
		{name: "s3 10gb", storageType: StorageTypeS3, dataSize: 10 * 1024 * 1024 * 1024},
		{name: "gcs 100gb", storageType: StorageTypeGCS, dataSize: 100 * 1024 * 1024 * 1024},
		{name: "azure 1tb", storageType: StorageTypeAzure, dataSize: 1024 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Enabled: true, StorageType: tt.storageType, Bucket: "test", Region: "us-east-1",
				Credentials: CredentialsConfig{UseIAMRole: true},
				Performance: PerformanceConfig{LocalCachePath: "/tmp"},
			}
			m, err := NewManager(cfg)
			require.NoError(t, err)

			estimate := m.EstimateCost(tt.dataSize)
			assert.Greater(t, estimate.DataSizeGB, float64(0))
			assert.Greater(t, estimate.StorageCostMonth, float64(0))
			assert.Greater(t, estimate.SavingsPercent, float64(0))
			assert.Greater(t, estimate.ComparedToEBS, estimate.StorageCostMonth)
		})
	}
}

func TestDefaultPoolCreateOptions(t *testing.T) {
	opts := DefaultPoolCreateOptions()

	assert.Equal(t, "12", opts.Properties["ashift"])
	assert.Equal(t, "on", opts.Properties["autoexpand"])
	assert.Equal(t, "lz4", opts.FSProperties["compression"])
	assert.Equal(t, "off", opts.FSProperties["atime"])
	assert.Equal(t, "128K", opts.FSProperties["recordsize"])
}

func TestDefaultTieringPolicy(t *testing.T) {
	policy := DefaultTieringPolicy()

	assert.Equal(t, 7*24*time.Hour, policy.ArchiveAfter)
	assert.Equal(t, 3, policy.KeepLocalCount)
	assert.Equal(t, 90*24*time.Hour, policy.DeleteArchivedAfter)
	assert.Equal(t, 6*time.Hour, policy.ScheduleInterval)
}

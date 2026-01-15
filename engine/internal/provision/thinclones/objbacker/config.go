/*
2025 © Postgres.ai
*/

// Package objbacker provides an interface to work with objbacker-backed ZFS pools.
// Objbacker is a native ZFS VDEV that talks directly to S3/GCS/Azure blob storage,
// enabling cost-effective tiered storage for database clones.
package objbacker

import (
	"fmt"
	"time"
)

// StorageType defines the type of object storage backend.
type StorageType string

const (
	// StorageTypeS3 represents Amazon S3 or S3-compatible storage.
	StorageTypeS3 StorageType = "s3"
	// StorageTypeGCS represents Google Cloud Storage.
	StorageTypeGCS StorageType = "gcs"
	// StorageTypeAzure represents Azure Blob Storage.
	StorageTypeAzure StorageType = "azure"
)

// Config defines configuration for objbacker-backed storage.
type Config struct {
	// Enabled determines if objbacker integration is active.
	Enabled bool `yaml:"enabled"`

	// StorageType specifies the object storage backend (s3, gcs, azure).
	StorageType StorageType `yaml:"storageType"`

	// Endpoint is the object storage endpoint URL.
	// For S3: https://s3.amazonaws.com or custom endpoint for MinIO/etc.
	// For GCS: https://storage.googleapis.com
	// For Azure: https://<account>.blob.core.windows.net
	Endpoint string `yaml:"endpoint"`

	// Bucket is the name of the bucket/container for storing data.
	Bucket string `yaml:"bucket"`

	// Prefix is the optional path prefix within the bucket.
	Prefix string `yaml:"prefix"`

	// Region is the cloud region (required for S3).
	Region string `yaml:"region"`

	// Credentials holds authentication configuration.
	Credentials CredentialsConfig `yaml:"credentials"`

	// Performance tuning options.
	Performance PerformanceConfig `yaml:"performance"`

	// Tiering configuration for hot/cold data separation.
	Tiering TieringConfig `yaml:"tiering"`

	// DevicePath is the path to the objbacker character device.
	// Defaults to /dev/zfs_objbacker.
	DevicePath string `yaml:"devicePath"`

	// DaemonSocketPath is the path to the objbacker daemon socket.
	DaemonSocketPath string `yaml:"daemonSocketPath"`
}

// CredentialsConfig holds authentication settings for object storage.
type CredentialsConfig struct {
	// AccessKeyID is the access key for S3/GCS.
	AccessKeyID string `yaml:"accessKeyId"`

	// SecretAccessKey is the secret key for S3/GCS.
	SecretAccessKey string `yaml:"secretAccessKey"`

	// CredentialsFile is the path to credentials file (for GCS service account).
	CredentialsFile string `yaml:"credentialsFile"`

	// UseIAMRole enables IAM role-based authentication (for cloud VMs).
	UseIAMRole bool `yaml:"useIamRole"`
}

// PerformanceConfig defines performance tuning parameters.
type PerformanceConfig struct {
	// BlockSize is the block size for object storage operations.
	// Larger blocks (1MB+) are more efficient for streaming.
	// Smaller blocks (<128KB) are better for random access.
	// Default: 1MB
	BlockSize int64 `yaml:"blockSize"`

	// LocalCacheSize is the size of the local NVMe cache in bytes.
	// Used for metadata and frequently accessed small blocks.
	// Default: 10GB
	LocalCacheSize int64 `yaml:"localCacheSize"`

	// LocalCachePath is the path to the local cache directory.
	// Should be on fast storage (NVMe).
	LocalCachePath string `yaml:"localCachePath"`

	// ReadAheadSize is the prefetch size for sequential reads.
	// Default: 4MB
	ReadAheadSize int64 `yaml:"readAheadSize"`

	// MaxConcurrentOps is the maximum number of concurrent object operations.
	// Default: 32
	MaxConcurrentOps int `yaml:"maxConcurrentOps"`

	// ConnectionTimeout is the timeout for establishing connections.
	ConnectionTimeout time.Duration `yaml:"connectionTimeout"`

	// RequestTimeout is the timeout for individual requests.
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

// TieringConfig defines data tiering behavior.
type TieringConfig struct {
	// Enabled determines if tiered storage is active.
	// When enabled, hot data stays on local storage while cold data
	// is automatically moved to object storage.
	Enabled bool `yaml:"enabled"`

	// HotDataThreshold is the age threshold for hot data in hours.
	// Data accessed more recently than this stays on local storage.
	// Default: 24 hours
	HotDataThreshold time.Duration `yaml:"hotDataThreshold"`

	// MetadataLocal keeps all metadata on local storage for faster access.
	// Default: true
	MetadataLocal bool `yaml:"metadataLocal"`

	// PromoteOnRead automatically promotes cold data to hot tier on read.
	// Default: true
	PromoteOnRead bool `yaml:"promoteOnRead"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:          false,
		StorageType:      StorageTypeS3,
		DevicePath:       "/dev/zfs_objbacker",
		DaemonSocketPath: "/var/run/zfs_objbacker.sock",
		Performance: PerformanceConfig{
			BlockSize:         1024 * 1024,             // 1MB
			LocalCacheSize:    10 * 1024 * 1024 * 1024, // 10GB
			LocalCachePath:    "/var/cache/objbacker",
			ReadAheadSize:     4 * 1024 * 1024, // 4MB
			MaxConcurrentOps:  32,
			ConnectionTimeout: 30 * time.Second,
			RequestTimeout:    5 * time.Minute,
		},
		Tiering: TieringConfig{
			Enabled:          true,
			HotDataThreshold: 24 * time.Hour,
			MetadataLocal:    true,
			PromoteOnRead:    true,
		},
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Bucket == "" {
		return fmt.Errorf("objbacker: bucket is required")
	}

	switch c.StorageType {
	case StorageTypeS3, StorageTypeGCS, StorageTypeAzure:
		// valid
	default:
		return fmt.Errorf("objbacker: invalid storage type: %s", c.StorageType)
	}

	if c.StorageType == StorageTypeS3 && c.Region == "" && !c.Credentials.UseIAMRole {
		return fmt.Errorf("objbacker: region is required for S3")
	}

	if !c.Credentials.UseIAMRole {
		if c.Credentials.AccessKeyID == "" && c.Credentials.CredentialsFile == "" {
			return fmt.Errorf("objbacker: credentials are required (accessKeyId or credentialsFile)")
		}
	}

	if c.Performance.LocalCachePath == "" {
		return fmt.Errorf("objbacker: localCachePath is required")
	}

	return nil
}

// ObjectPath returns the full object path for a given key.
func (c *Config) ObjectPath(key string) string {
	if c.Prefix != "" {
		return fmt.Sprintf("%s/%s", c.Prefix, key)
	}
	return key
}

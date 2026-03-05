/*
2025 © PostgresAI
*/

// Package rdsrefresh provides RDS/Aurora refresh functionality for DBLab Engine.
package rdsrefresh

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultPollInterval = 30 * time.Second
	defaultTimeout      = 4 * time.Hour
	defaultMaxCloneAge  = 48 * time.Hour

	sourceTypeRDS           = "rds"
	sourceTypeAuroraCluster = "aurora-cluster"

	// ManagedByTagKey is the tag key used to identify clones managed by this tool.
	ManagedByTagKey = "ManagedBy"
	// ManagedByTagValue is the tag value used to identify clones managed by this tool.
	ManagedByTagValue = "dblab-rds-refresh"
	// AutoDeleteTagKey is the tag key used to mark clones for auto-deletion.
	AutoDeleteTagKey = "AutoDelete"
	// DeleteAfterTagKey is the tag key containing the timestamp after which the clone should be deleted.
	DeleteAfterTagKey = "DeleteAfter"
)

// Config holds the configuration for the RDS refresh component.
type Config struct {
	Source   SourceConfig   `yaml:"source"`
	RDSClone RDSCloneConfig `yaml:"clone"`
	DBLab    DBLabConfig    `yaml:"dblab"`
	AWS      AWSConfig      `yaml:"aws"`
}

// SourceConfig defines the source RDS/Aurora database to clone from.
type SourceConfig struct {
	// Type specifies the source type: "rds" for RDS instance, "aurora-cluster" for Aurora cluster.
	Type string `yaml:"type"`
	// Identifier is the RDS DB instance identifier or Aurora cluster identifier.
	Identifier string `yaml:"identifier"`
	// SnapshotIdentifier is the specific snapshot to use. If empty, the latest automated snapshot is used.
	SnapshotIdentifier string `yaml:"snapshotIdentifier"`
	// DBName is the database name to connect to (used when updating DBLab config).
	DBName string `yaml:"dbName"`
	// Username is the database username (used when updating DBLab config).
	Username string `yaml:"username"`
	// Password is the database password (used when updating DBLab config).
	Password string `yaml:"password"`
}

// RDSCloneConfig defines settings for the temporary RDS clone.
type RDSCloneConfig struct {
	// InstanceClass is the DB instance class for the clone (e.g., "db.t3.medium").
	InstanceClass string `yaml:"instanceClass"`
	// DBSubnetGroupName is the DB subnet group for the clone.
	DBSubnetGroupName string `yaml:"subnetGroup"`
	// VPCSecurityGroupIDs are the security group IDs to assign to the clone.
	VPCSecurityGroupIDs []string `yaml:"securityGroups"`
	// PubliclyAccessible determines if the clone should be publicly accessible.
	PubliclyAccessible bool `yaml:"publiclyAccessible"`
	// Tags are additional tags to add to the clone.
	Tags map[string]string `yaml:"tags"`
	// ParameterGroupName is the parameter group to use for the clone.
	ParameterGroupName string `yaml:"parameterGroup"`
	// OptionGroupName is the option group to use for the clone (RDS only).
	OptionGroupName string `yaml:"optionGroup"`
	// DBClusterParameterGroupName is the cluster parameter group for Aurora clones.
	DBClusterParameterGroupName string `yaml:"clusterParameterGroup"`
	// Port is the port for the clone. If 0, uses default port.
	Port int32 `yaml:"port"`
	// EnableIAMAuth enables IAM database authentication.
	EnableIAMAuth bool `yaml:"enableIAMAuth"`
	// StorageType specifies storage type (gp2, gp3, io1, etc.) for RDS clones.
	StorageType string `yaml:"storageType"`
	// DeletionProtection enables deletion protection on the clone.
	DeletionProtection bool `yaml:"deletionProtection"`
	// MaxAge is the maximum age of a clone before it's considered stale and eligible for cleanup.
	MaxAge Duration `yaml:"maxAge"`
}

// DBLabConfig defines the DBLab Engine connection settings.
type DBLabConfig struct {
	// APIEndpoint is the DBLab Engine API endpoint (e.g., "https://dblab.example.com:2345").
	APIEndpoint string `yaml:"apiEndpoint"`
	// Token is the verification token for the DBLab API.
	Token string `yaml:"token"`
	// Insecure allows connections to DBLab with invalid TLS certificates.
	Insecure bool `yaml:"insecure"`
	// PollInterval is how often to poll the DBLab status during refresh.
	PollInterval Duration `yaml:"pollInterval"`
	// Timeout is the maximum time to wait for the refresh to complete.
	Timeout Duration `yaml:"timeout"`
}

// AWSConfig holds AWS-specific settings.
type AWSConfig struct {
	// Region is the AWS region where the RDS/Aurora resources are located.
	Region string `yaml:"region"`
	// Endpoint is a custom AWS endpoint (useful for testing with LocalStack).
	Endpoint string `yaml:"endpoint"`
}

// Duration is a wrapper around time.Duration for YAML parsing.
type Duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler for Duration.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}

	*d = Duration(dur)

	return nil
}

// MarshalYAML implements yaml.Marshaler for Duration.
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// Duration returns the time.Duration value.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// expand environment variables in the config
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.SetDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.Source.Type == "" {
		return fmt.Errorf("source.type is required (rds or aurora-cluster)")
	}

	if c.Source.Type != sourceTypeRDS && c.Source.Type != sourceTypeAuroraCluster {
		return fmt.Errorf("source.type must be %q or %q, got %q", sourceTypeRDS, sourceTypeAuroraCluster, c.Source.Type)
	}

	if c.Source.Identifier == "" {
		return fmt.Errorf("source.identifier is required")
	}

	if c.Source.DBName == "" {
		return fmt.Errorf("source.dbName is required")
	}

	if c.Source.Username == "" {
		return fmt.Errorf("source.username is required")
	}

	if c.Source.Password == "" {
		return fmt.Errorf("source.password is required")
	}

	if c.RDSClone.InstanceClass == "" {
		return fmt.Errorf("clone.instanceClass is required")
	}

	if c.DBLab.APIEndpoint == "" {
		return fmt.Errorf("dblab.apiEndpoint is required")
	}

	if c.DBLab.Token == "" {
		return fmt.Errorf("dblab.token is required")
	}

	if c.AWS.Region == "" {
		return fmt.Errorf("aws.region is required")
	}

	return nil
}

// SetDefaults sets default values for optional configuration fields.
func (c *Config) SetDefaults() {
	if c.DBLab.PollInterval == 0 {
		c.DBLab.PollInterval = Duration(defaultPollInterval)
	}

	if c.DBLab.Timeout == 0 {
		c.DBLab.Timeout = Duration(defaultTimeout)
	}

	if c.RDSClone.MaxAge == 0 {
		c.RDSClone.MaxAge = Duration(defaultMaxCloneAge)
	}

	if c.RDSClone.Tags == nil {
		c.RDSClone.Tags = make(map[string]string)
	}

	c.RDSClone.Tags[ManagedByTagKey] = ManagedByTagValue
	c.RDSClone.Tags[AutoDeleteTagKey] = "true"
}

// GetDeleteAfterTime returns the time after which the clone should be deleted.
func (c *Config) GetDeleteAfterTime() time.Time {
	return time.Now().UTC().Add(c.RDSClone.MaxAge.Duration())
}

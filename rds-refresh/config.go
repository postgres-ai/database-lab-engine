/*
2024 © Postgres.ai
*/

package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration for the RDS refresh component.
type Config struct {
	Source SourceConfig `yaml:"source"`
	Clone  CloneConfig  `yaml:"clone"`
	DBLab  DBLabConfig  `yaml:"dblab"`
	AWS    AWSConfig    `yaml:"aws"`
}

// SourceConfig defines the source RDS/Aurora database to clone from.
type SourceConfig struct {
	// Type specifies the source type: "rds" for RDS instance, "aurora-cluster" for Aurora cluster.
	Type string `yaml:"type"`
	// Identifier is the RDS DB instance identifier or Aurora cluster identifier.
	Identifier string `yaml:"identifier"`
	// SnapshotIdentifier is the specific snapshot to use. If empty, the latest automated snapshot is used.
	SnapshotIdentifier string `yaml:"snapshotIdentifier"`
}

// CloneConfig defines settings for the temporary clone.
type CloneConfig struct {
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

	// Expand environment variables in the config
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	cfg.SetDefaults()

	return &cfg, nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.Source.Type == "" {
		return fmt.Errorf("source.type is required (rds or aurora-cluster)")
	}

	if c.Source.Type != "rds" && c.Source.Type != "aurora-cluster" {
		return fmt.Errorf("source.type must be 'rds' or 'aurora-cluster', got %q", c.Source.Type)
	}

	if c.Source.Identifier == "" {
		return fmt.Errorf("source.identifier is required")
	}

	if c.Clone.InstanceClass == "" {
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
		c.DBLab.PollInterval = Duration(30 * time.Second)
	}

	if c.DBLab.Timeout == 0 {
		c.DBLab.Timeout = Duration(4 * time.Hour)
	}

	if c.Clone.Tags == nil {
		c.Clone.Tags = make(map[string]string)
	}

	c.Clone.Tags["ManagedBy"] = "dblab-rds-refresh"
	c.Clone.Tags["AutoDelete"] = "true"
}

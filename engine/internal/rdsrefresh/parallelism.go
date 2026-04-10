/*
2026 © PostgresAI
*/

package rdsrefresh

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	// rdsInstanceClassPrefix is stripped to derive the EC2 instance type.
	rdsInstanceClassPrefix = "db."

	// minParallelJobs is the minimum parallelism level.
	minParallelJobs = 1
)

// EC2API defines the interface for EC2 client operations used for vCPU lookup.
type EC2API interface {
	DescribeInstanceTypes(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error)
}

// ParallelismConfig holds the computed parallelism levels for dump and restore.
type ParallelismConfig struct {
	DumpJobs    int
	RestoreJobs int
}

// ResolveParallelism determines the optimal parallelism levels for pg_dump and pg_restore.
// dump parallelism is based on the vCPU count of the RDS clone instance class.
// restore parallelism is based on the vCPU count of the local machine.
func ResolveParallelism(ctx context.Context, cfg *Config) (*ParallelismConfig, error) {
	dumpJobs, err := resolveRDSInstanceVCPUs(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve RDS instance vCPUs: %w", err)
	}

	restoreJobs := resolveLocalVCPUs()

	log.Msg("auto-parallelism: dump jobs =", dumpJobs, "(RDS clone vCPUs), restore jobs =", restoreJobs, "(local vCPUs)")

	return &ParallelismConfig{
		DumpJobs:    dumpJobs,
		RestoreJobs: restoreJobs,
	}, nil
}

// resolveRDSInstanceVCPUs looks up the vCPU count for the configured RDS instance class
// by querying the EC2 DescribeInstanceTypes API.
func resolveRDSInstanceVCPUs(ctx context.Context, cfg *Config) (int, error) {
	ec2Client, err := newEC2Client(ctx, cfg)
	if err != nil {
		return 0, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	return lookupInstanceVCPUs(ctx, ec2Client, cfg.RDSClone.InstanceClass)
}

func newEC2Client(ctx context.Context, cfg *Config) (EC2API, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWS.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var opts []func(*ec2.Options)
	if cfg.AWS.Endpoint != "" {
		opts = append(opts, func(o *ec2.Options) {
			o.BaseEndpoint = aws.String(cfg.AWS.Endpoint)
		})
	}

	return ec2.NewFromConfig(awsCfg, opts...), nil
}

// lookupInstanceVCPUs queries EC2 for the vCPU count of the given RDS instance class.
func lookupInstanceVCPUs(ctx context.Context, client EC2API, rdsInstanceClass string) (int, error) {
	ec2InstanceType, err := rdsClassToEC2Type(rdsInstanceClass)
	if err != nil {
		return 0, err
	}

	result, err := client.DescribeInstanceTypes(ctx, &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []ec2types.InstanceType{ec2types.InstanceType(ec2InstanceType)},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to describe EC2 instance type %q: %w", ec2InstanceType, err)
	}

	if len(result.InstanceTypes) == 0 {
		return 0, fmt.Errorf("EC2 instance type %q not found", ec2InstanceType)
	}

	info := result.InstanceTypes[0]
	if info.VCpuInfo == nil || info.VCpuInfo.DefaultVCpus == nil {
		return 0, fmt.Errorf("vCPU info not available for instance type %q", ec2InstanceType)
	}

	vcpus := int(*info.VCpuInfo.DefaultVCpus)
	if vcpus < minParallelJobs {
		return minParallelJobs, nil
	}

	return vcpus, nil
}

// rdsClassToEC2Type converts an RDS instance class (e.g. "db.m5.xlarge") to an EC2 instance type ("m5.xlarge").
func rdsClassToEC2Type(rdsClass string) (string, error) {
	if !strings.HasPrefix(rdsClass, rdsInstanceClassPrefix) {
		return "", fmt.Errorf("invalid RDS instance class %q: expected %q prefix", rdsClass, rdsInstanceClassPrefix)
	}

	ec2Type := strings.TrimPrefix(rdsClass, rdsInstanceClassPrefix)
	if ec2Type == "" {
		return "", fmt.Errorf("invalid RDS instance class %q: empty after removing prefix", rdsClass)
	}

	return ec2Type, nil
}

// resolveLocalVCPUs returns the number of logical CPUs available on the local machine.
func resolveLocalVCPUs() int {
	cpus := runtime.NumCPU()
	if cpus < minParallelJobs {
		return minParallelJobs
	}

	return cpus
}

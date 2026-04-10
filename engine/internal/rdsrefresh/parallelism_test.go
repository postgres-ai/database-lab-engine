/*
2026 © PostgresAI
*/

package rdsrefresh

import (
	"context"
	"runtime"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEC2API struct {
	describeInstanceTypesFunc func(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error)
}

func (m *mockEC2API) DescribeInstanceTypes(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
	if m.describeInstanceTypesFunc != nil {
		return m.describeInstanceTypesFunc(ctx, params, optFns...)
	}

	return &ec2.DescribeInstanceTypesOutput{}, nil
}

func TestRdsClassToEC2Type(t *testing.T) {
	testCases := []struct {
		rdsClass     string
		expectedType string
		expectErr    bool
	}{
		{rdsClass: "db.m5.xlarge", expectedType: "m5.xlarge"},
		{rdsClass: "db.t3.medium", expectedType: "t3.medium"},
		{rdsClass: "db.r6g.2xlarge", expectedType: "r6g.2xlarge"},
		{rdsClass: "db.serverless", expectedType: "serverless"},
		{rdsClass: "m5.xlarge", expectErr: true},
		{rdsClass: "db.", expectErr: true},
		{rdsClass: "", expectErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.rdsClass, func(t *testing.T) {
			result, err := rdsClassToEC2Type(tc.rdsClass)

			if tc.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedType, result)
		})
	}
}

func TestLookupInstanceVCPUs(t *testing.T) {
	t.Run("returns vcpu count for valid instance type", func(t *testing.T) {
		mock := &mockEC2API{
			describeInstanceTypesFunc: func(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
				assert.Equal(t, ec2types.InstanceType("m5.xlarge"), params.InstanceTypes[0])

				return &ec2.DescribeInstanceTypesOutput{
					InstanceTypes: []ec2types.InstanceTypeInfo{
						{InstanceType: ec2types.InstanceType("m5.xlarge"), VCpuInfo: &ec2types.VCpuInfo{DefaultVCpus: aws.Int32(4)}},
					},
				}, nil
			},
		}

		vcpus, err := lookupInstanceVCPUs(context.Background(), mock, "db.m5.xlarge")

		require.NoError(t, err)
		assert.Equal(t, 4, vcpus)
	})

	t.Run("returns vcpu count for large instance", func(t *testing.T) {
		mock := &mockEC2API{
			describeInstanceTypesFunc: func(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
				return &ec2.DescribeInstanceTypesOutput{
					InstanceTypes: []ec2types.InstanceTypeInfo{
						{InstanceType: ec2types.InstanceType("r6g.16xlarge"), VCpuInfo: &ec2types.VCpuInfo{DefaultVCpus: aws.Int32(64)}},
					},
				}, nil
			},
		}

		vcpus, err := lookupInstanceVCPUs(context.Background(), mock, "db.r6g.16xlarge")

		require.NoError(t, err)
		assert.Equal(t, 64, vcpus)
	})

	t.Run("returns error for instance type not found", func(t *testing.T) {
		mock := &mockEC2API{
			describeInstanceTypesFunc: func(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
				return &ec2.DescribeInstanceTypesOutput{InstanceTypes: []ec2types.InstanceTypeInfo{}}, nil
			},
		}

		_, err := lookupInstanceVCPUs(context.Background(), mock, "db.nonexistent.type")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns error for missing vcpu info", func(t *testing.T) {
		mock := &mockEC2API{
			describeInstanceTypesFunc: func(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
				return &ec2.DescribeInstanceTypesOutput{
					InstanceTypes: []ec2types.InstanceTypeInfo{
						{InstanceType: ec2types.InstanceType("m5.xlarge"), VCpuInfo: nil},
					},
				}, nil
			},
		}

		_, err := lookupInstanceVCPUs(context.Background(), mock, "db.m5.xlarge")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "vCPU info not available")
	})

	t.Run("returns error for invalid rds class", func(t *testing.T) {
		mock := &mockEC2API{}
		_, err := lookupInstanceVCPUs(context.Background(), mock, "invalid-class")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid RDS instance class")
	})

	t.Run("returns error on api failure", func(t *testing.T) {
		mock := &mockEC2API{
			describeInstanceTypesFunc: func(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
				return nil, assert.AnError
			},
		}

		_, err := lookupInstanceVCPUs(context.Background(), mock, "db.m5.xlarge")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to describe EC2 instance type")
	})
}

func TestResolveLocalVCPUs(t *testing.T) {
	vcpus := resolveLocalVCPUs()

	assert.Equal(t, runtime.NumCPU(), vcpus)
	assert.GreaterOrEqual(t, vcpus, minParallelJobs)
}

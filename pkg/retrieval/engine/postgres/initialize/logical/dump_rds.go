/*
2020 © Postgres.ai
*/

package logical

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
)

const (
	sslRootCert = "/tmp/cert/ssl-combined-ca-bundle.pem"

	// AWS service names.
	serviceIAM   = "iam"
	serviceRDSDB = "rds-db"

	// RDS policy option values.
	rdsDocPolicyVersion = "2012-10-17"
	rdsDBConnectAction  = "rds-db:connect"
)

type rdsDumper struct {
	rdsCfg *RDSConfig
	iamSvc *iam.IAM
	rdsSvc *rds.RDS
}

// RDSConfig describes configuration of an RDS instance.
type RDSConfig struct {
	IamPolicyName string `yaml:"iamPolicyName"`
	AWSRegion     string `yaml:"awsRegion"`
	DBInstance    string `yaml:"dbInstanceIdentifier"`
	Username      string `yaml:"username"`
	SSLRootCert   string `yaml:"sslRootCert"`
}

type policyDocument struct {
	Version   string            `json:"Version"`
	Statement []policyStatement `json:"Statement"`
}

type policyStatement struct {
	Effect   string   `json:"Effect"`
	Action   []string `json:"Action"`
	Resource []string `json:"Resource"`
}

func newRDSDumper(rdsCfg *RDSConfig) (*rdsDumper, error) {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(rdsCfg.AWSRegion),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start AWS session")
	}

	return &rdsDumper{
		rdsCfg: rdsCfg,
		iamSvc: iam.New(awsSession, aws.NewConfig()),
		rdsSvc: rds.New(awsSession),
	}, nil
}

// GetEnvVariables returns dumper environment variables.
func (r *rdsDumper) GetCmdEnvVariables() []string {
	execEnvs := []string{
		"PGSSLROOTCERT=" + sslRootCert,
		"PGSSLMODE=verify-ca",
	}

	return execEnvs
}

// GetMounts returns dumper volume configurations for mounting.
func (r *rdsDumper) GetMounts() []mount.Mount {
	mounts := []mount.Mount{}

	if r.rdsCfg != nil && r.rdsCfg.SSLRootCert != "" {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: r.rdsCfg.SSLRootCert,
			Target: sslRootCert,
		})
	}

	return mounts
}

// SetConnectionOptions sets connection options for dumping.
func (r *rdsDumper) SetConnectionOptions(ctx context.Context, c *Connection) error {
	dbInstancesOutput, err := r.rdsSvc.DescribeDBInstancesWithContext(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(r.rdsCfg.DBInstance),
	})
	if err != nil {
		return err
	}

	const expectedDBInstances = 1

	if len(dbInstancesOutput.DBInstances) != expectedDBInstances {
		return errors.Errorf("invalid quantity of DBInstances: %d", len(dbInstancesOutput.DBInstances))
	}

	dbInstance := dbInstancesOutput.DBInstances[0]

	dbARN := dbInstancesOutput.DBInstances[0].DBInstanceArn
	parsedDBArn, err := arn.Parse(aws.StringValue(dbARN))

	if err != nil {
		return errors.Wrap(err, "failed to parse a database ARN")
	}

	policy, err := r.getPolicy(r.getPolicyARN(parsedDBArn).String(), parsedDBArn, dbInstance)
	if err != nil {
		return errors.Wrap(err, "failed to get a policy")
	}

	_, err = r.iamSvc.AttachUserPolicy(&iam.AttachUserPolicyInput{
		PolicyArn: policy.Arn,
		UserName:  aws.String(r.rdsCfg.Username),
	})
	if err != nil {
		return errors.Wrap(err, "failed to attach policy to a user")
	}

	log.Msg(fmt.Sprintf("Policy %q has been attached to %s", aws.StringValue(policy.Arn), r.rdsCfg.Username))

	dbAuthToken, err := rdsutils.BuildAuthToken(
		fmt.Sprintf("%s:%d", aws.StringValue(dbInstance.Endpoint.Address), int(aws.Int64Value(dbInstance.Endpoint.Port))),
		r.rdsCfg.AWSRegion,
		c.Username,
		r.rdsSvc.Config.Credentials,
	)
	if err != nil {
		return errors.Wrap(err, "failed to build an auth token")
	}

	c.Host = aws.StringValue(dbInstance.Endpoint.Address)
	c.Port = int(aws.Int64Value(dbInstance.Endpoint.Port))
	c.Password = dbAuthToken

	return nil
}

func (r *rdsDumper) getPolicyARN(parsedDBArn arn.ARN) arn.ARN {
	policyARN := arn.ARN{
		Partition: parsedDBArn.Partition,
		Service:   serviceIAM,
		AccountID: parsedDBArn.AccountID,
		Resource:  "policy/" + r.rdsCfg.IamPolicyName,
	}

	return policyARN
}

func (r *rdsDumper) getPolicy(policyARN string, parsedDBArn arn.ARN, dbInstance *rds.DBInstance) (*iam.Policy, error) {
	policyOutput, err := r.iamSvc.GetPolicy(&iam.GetPolicyInput{
		PolicyArn: aws.String(policyARN),
	})
	if err == nil {
		return policyOutput.Policy, nil
	}

	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != iam.ErrCodeNoSuchEntityException {
		return nil, errors.Wrap(err, "failed to get policy")
	}

	policyDocumentData, err := json.Marshal(r.buildPolicyDocument(parsedDBArn, dbInstance))
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal a policy document")
	}

	createPolicyOutput, err := r.iamSvc.CreatePolicy(&iam.CreatePolicyInput{
		PolicyDocument: aws.String(string(policyDocumentData)),
		PolicyName:     aws.String(r.rdsCfg.IamPolicyName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a policy")
	}

	log.Msg(fmt.Sprintf("Policy has been created: %v", aws.StringValue(createPolicyOutput.Policy.Arn)))

	return createPolicyOutput.Policy, nil
}

func (r *rdsDumper) buildPolicyDocument(parsedDBArn arn.ARN, dbInstance *rds.DBInstance) policyDocument {
	dbPolicyARN := arn.ARN{
		Partition: parsedDBArn.Partition,
		Service:   serviceRDSDB,
		Region:    parsedDBArn.Region,
		AccountID: parsedDBArn.AccountID,
		Resource:  fmt.Sprintf("dbuser:%s/*", aws.StringValue(dbInstance.DbiResourceId)),
	}

	policyDocument := policyDocument{
		Version: rdsDocPolicyVersion,
		Statement: []policyStatement{
			{
				Effect:   "Allow",
				Action:   []string{rdsDBConnectAction},
				Resource: []string{dbPolicyARN.String()},
			},
		},
	}

	return policyDocument
}

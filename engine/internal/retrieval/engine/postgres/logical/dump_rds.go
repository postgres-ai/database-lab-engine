/*
2020 © Postgres.ai
*/

package logical

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	sslRootCert = "/cert/ssl-combined-ca-bundle.pem"
)

type rdsDumper struct {
	rdsCfg *RDSConfig
	iamSvc *iam.IAM
	rdsSvc *rds.RDS
}

// RDSConfig describes configuration of an RDS instance.
type RDSConfig struct {
	AWSRegion   string `yaml:"awsRegion"`
	DBInstance  string `yaml:"dbInstanceIdentifier"`
	SSLRootCert string `yaml:"sslRootCert"`
}

func newRDSDumper(rdsCfg *RDSConfig) (*rdsDumper, error) {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(rdsCfg.AWSRegion),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start AWS session")
	}

	credentials, err := awsSession.Config.Credentials.Get()
	if err != nil || !credentials.HasKeys() {
		log.Dbg(err)

		return nil, errors.New(`failed to check AWS credentials.
Set up valid environment variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY`)
	}

	return &rdsDumper{
		rdsCfg: rdsCfg,
		iamSvc: iam.New(awsSession, aws.NewConfig()),
		rdsSvc: rds.New(awsSession),
	}, nil
}

// GetEnvVariables returns dumper environment variables.
func (r *rdsDumper) GetCmdEnvVariables() []string {
	cert := sslRootCert

	if r.rdsCfg.SSLRootCert != "" {
		cert = r.rdsCfg.SSLRootCert
	}

	execEnvs := []string{
		"PGSSLROOTCERT=" + cert,
		"PGSSLMODE=verify-ca",
	}

	return execEnvs
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

func (r *rdsDumper) GetDatabaseListQuery() string {
	return "select datname from pg_catalog.pg_database where not datistemplate"
}

/*
2025 © PostgresAI
*/

package rdsrefresh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsrds "github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/projection"
)

func TestRefreshResult(t *testing.T) {
	t.Run("successful result", func(t *testing.T) {
		start := time.Now()
		end := start.Add(time.Hour)
		result := &RefreshResult{
			Success:       true,
			SnapshotID:    "snap-123",
			CloneID:       "clone-456",
			StartTime:     start,
			EndTime:       end,
			Duration:      end.Sub(start),
			CloneEndpoint: "clone.rds.amazonaws.com",
		}

		assert.True(t, result.Success)
		assert.Equal(t, "snap-123", result.SnapshotID)
		assert.Equal(t, "clone-456", result.CloneID)
		assert.Equal(t, time.Hour, result.Duration)
		assert.NoError(t, result.Error)
	})

	t.Run("failed result with error", func(t *testing.T) {
		result := &RefreshResult{Success: false, Error: assert.AnError}

		assert.False(t, result.Success)
		assert.Error(t, result.Error)
	})
}

func TestDryRun(t *testing.T) {
	t.Run("performs validation checks", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
				return
			}

			if r.URL.Path == "/status" {
				status := &models.InstanceStatus{Retrieving: models.Retrieving{Status: models.Finished}}
				_ = json.NewEncoder(w).Encode(status)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		cfg := &Config{
			Source:   SourceConfig{Type: "rds", Identifier: "test-db", DBName: "postgres", Username: "postgres", Password: "secret"},
			RDSClone: RDSCloneConfig{InstanceClass: "db.t3.medium"},
			DBLab:    DBLabConfig{APIEndpoint: server.URL, Token: "test-token"},
			AWS:      AWSConfig{Region: "us-east-1"},
		}

		refresher, err := NewRefresher(context.Background(), cfg)
		if err != nil {
			t.Skip("skipping test due to AWS credentials requirement")
		}

		err = refresher.DryRun(context.Background())
		if err != nil {
			assert.Contains(t, err.Error(), "source")
		}
	})
}

func TestRefresherCreation(t *testing.T) {
	t.Run("creates refresher with valid config", func(t *testing.T) {
		cfg := &Config{
			Source:   SourceConfig{Type: "rds", Identifier: "test-db", DBName: "postgres", Username: "postgres", Password: "secret"},
			RDSClone: RDSCloneConfig{InstanceClass: "db.t3.medium"},
			DBLab:    DBLabConfig{APIEndpoint: "https://dblab:2345", Token: "test-token"},
			AWS:      AWSConfig{Region: "us-east-1"},
		}

		refresher, err := NewRefresher(context.Background(), cfg)
		if err != nil {
			t.Skip("skipping test due to AWS credentials requirement")
		}

		assert.NotNil(t, refresher)
		assert.NotNil(t, refresher.cfg)
		assert.NotNil(t, refresher.rds)
		assert.NotNil(t, refresher.dblab)
	})

	t.Run("fails with invalid dblab endpoint", func(t *testing.T) {
		cfg := &Config{
			Source:   SourceConfig{Type: "rds", Identifier: "test-db", DBName: "postgres", Username: "postgres", Password: "secret"},
			RDSClone: RDSCloneConfig{InstanceClass: "db.t3.medium"},
			DBLab:    DBLabConfig{APIEndpoint: "://invalid-url", Token: "test-token"},
			AWS:      AWSConfig{Region: "us-east-1"},
		}

		_, err := NewRefresher(context.Background(), cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create DBLab client")
	})
}

func TestRefreshWorkflow(t *testing.T) {
	t.Run("early exit when refresh already in progress", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
				return
			}

			if r.URL.Path == "/status" {
				status := &models.InstanceStatus{Retrieving: models.Retrieving{Status: models.Refreshing}}
				_ = json.NewEncoder(w).Encode(status)
				return
			}
		}))
		defer server.Close()

		cfg := &Config{
			Source:   SourceConfig{Type: "rds", Identifier: "test-db", DBName: "postgres", Username: "postgres", Password: "secret"},
			RDSClone: RDSCloneConfig{InstanceClass: "db.t3.medium"},
			DBLab:    DBLabConfig{APIEndpoint: server.URL, Token: "test-token"},
			AWS:      AWSConfig{Region: "us-east-1"},
		}

		refresher, err := NewRefresher(context.Background(), cfg)
		if err != nil {
			t.Skip("skipping test due to AWS credentials requirement")
		}

		result := refresher.Run(context.Background())
		assert.False(t, result.Success)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "refresh already in progress")
	})

	t.Run("early exit on dblab health check failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		cfg := &Config{
			Source:   SourceConfig{Type: "rds", Identifier: "test-db", DBName: "postgres", Username: "postgres", Password: "secret"},
			RDSClone: RDSCloneConfig{InstanceClass: "db.t3.medium"},
			DBLab:    DBLabConfig{APIEndpoint: server.URL, Token: "test-token"},
			AWS:      AWSConfig{Region: "us-east-1"},
		}

		refresher, err := NewRefresher(context.Background(), cfg)
		if err != nil {
			t.Skip("skipping test due to AWS credentials requirement")
		}

		result := refresher.Run(context.Background())
		assert.False(t, result.Success)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "DBLab health check failed")
	})

	t.Run("calculates duration correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		cfg := &Config{
			Source:   SourceConfig{Type: "rds", Identifier: "test-db", DBName: "postgres", Username: "postgres", Password: "secret"},
			RDSClone: RDSCloneConfig{InstanceClass: "db.t3.medium"},
			DBLab:    DBLabConfig{APIEndpoint: server.URL, Token: "test-token"},
			AWS:      AWSConfig{Region: "us-east-1"},
		}

		refresher, err := NewRefresher(context.Background(), cfg)
		if err != nil {
			t.Skip("skipping test due to AWS credentials requirement")
		}

		start := time.Now()
		result := refresher.Run(context.Background())
		elapsed := time.Since(start)

		assert.False(t, result.Success)
		assert.GreaterOrEqual(t, result.Duration, time.Duration(0))
		assert.LessOrEqual(t, result.Duration, elapsed+100*time.Millisecond)
	})
}

func TestRefreshWorkflowPassesCloneIdentifier(t *testing.T) {
	t.Run("passes clone identifier as rds iam db instance", func(t *testing.T) {
		var receivedConfig models.ConfigProjection

		statusCallCount := 0
		refreshTriggered := false

		dblabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/healthz":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			case "/status":
				statusCallCount++

				var status models.RetrievalStatus

				switch {
				case !refreshTriggered || statusCallCount <= 2:
					status = models.Finished
				case statusCallCount == 3:
					status = models.Refreshing
				default:
					status = models.Finished
				}

				resp := &models.InstanceStatus{Retrieving: models.Retrieving{Status: status}}
				_ = json.NewEncoder(w).Encode(resp)
			case "/admin/config":
				var nested map[string]interface{}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&nested))
				require.NoError(t, projection.LoadJSON(&receivedConfig, nested, projection.LoadOptions{Groups: []string{"default", "sensitive"}}))
				w.WriteHeader(http.StatusOK)
			case "/full-refresh":
				refreshTriggered = true
				statusCallCount = 0
				_, _ = w.Write([]byte(`{"status":"OK"}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer dblabServer.Close()

		mock := &mockRDSAPI{
			describeDBInstancesFunc: func(ctx context.Context, params *awsrds.DescribeDBInstancesInput, optFns ...func(*awsrds.Options)) (*awsrds.DescribeDBInstancesOutput, error) {
				return &awsrds.DescribeDBInstancesOutput{
					DBInstances: []types.DBInstance{{
						DBInstanceIdentifier: aws.String(aws.ToString(params.DBInstanceIdentifier)),
						DBInstanceStatus:     aws.String("available"),
						Engine:               aws.String("postgres"),
						EngineVersion:        aws.String("15.4"),
						Endpoint:             &types.Endpoint{Address: aws.String("clone.rds.amazonaws.com"), Port: aws.Int32(5432)},
					}},
				}, nil
			},
			restoreDBInstanceFunc: func(ctx context.Context, params *awsrds.RestoreDBInstanceFromDBSnapshotInput, optFns ...func(*awsrds.Options)) (*awsrds.RestoreDBInstanceFromDBSnapshotOutput, error) {
				return &awsrds.RestoreDBInstanceFromDBSnapshotOutput{}, nil
			},
			modifyDBInstanceFunc: func(ctx context.Context, params *awsrds.ModifyDBInstanceInput, optFns ...func(*awsrds.Options)) (*awsrds.ModifyDBInstanceOutput, error) {
				return &awsrds.ModifyDBInstanceOutput{}, nil
			},
			deleteDBInstanceFunc: func(ctx context.Context, params *awsrds.DeleteDBInstanceInput, optFns ...func(*awsrds.Options)) (*awsrds.DeleteDBInstanceOutput, error) {
				return &awsrds.DeleteDBInstanceOutput{}, nil
			},
		}

		cfg := &Config{
			Source:   SourceConfig{Type: "rds", Identifier: "test-db", DBName: "postgres", Username: "dbuser", Password: "dbpass", SnapshotIdentifier: "snap-123"},
			RDSClone: RDSCloneConfig{InstanceClass: "db.t3.medium"},
			DBLab:    DBLabConfig{APIEndpoint: dblabServer.URL, Token: "test-token", PollInterval: Duration(100 * time.Millisecond), Timeout: Duration(5 * time.Second)},
		}

		rdsClient := NewRDSClientWithAPI(mock, cfg)
		dblabClient, err := NewDBLabClient(&cfg.DBLab)
		require.NoError(t, err)

		refresher := &Refresher{cfg: cfg, rds: rdsClient, dblab: dblabClient}
		result := refresher.Run(context.Background())

		require.True(t, result.Success, "expected success but got error: %v", result.Error)
		require.NotNil(t, receivedConfig.RDSIAMDBInstance, "RDSIAMDBInstance should be set in config update")
		assert.Contains(t, *receivedConfig.RDSIAMDBInstance, cloneNamePrefix)
		assert.Equal(t, "clone.rds.amazonaws.com", *receivedConfig.Host)
		assert.Equal(t, int64(5432), *receivedConfig.Port)
		assert.Equal(t, "postgres", *receivedConfig.DBName)
		assert.Equal(t, "dbuser", *receivedConfig.Username)
		assert.Equal(t, "dbpass", *receivedConfig.Password)
	})
}

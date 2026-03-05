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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
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

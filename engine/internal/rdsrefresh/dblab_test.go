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

func TestDBLabClientHealth(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/healthz", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.Health(context.Background())
		assert.NoError(t, err)
	})

	t.Run("unhealthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.Health(context.Background())
		assert.Error(t, err)
	})
}

func TestDBLabClientGetStatus(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		expectedStatus := &models.InstanceStatus{
			Retrieving: models.Retrieving{Status: models.Finished},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/status", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedStatus)
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		status, err := client.GetStatus(context.Background())
		require.NoError(t, err)
		assert.Equal(t, models.Finished, status.Retrieving.Status)
	})
}

func TestDBLabClientIsRefreshInProgress(t *testing.T) {
	testCases := []struct {
		name           string
		status         models.RetrievalStatus
		expectedResult bool
	}{
		{name: "refreshing", status: models.Refreshing, expectedResult: true},
		{name: "snapshotting", status: models.Snapshotting, expectedResult: true},
		{name: "pending", status: models.Pending, expectedResult: true},
		{name: "renewed", status: models.Renewed, expectedResult: true},
		{name: "finished", status: models.Finished, expectedResult: false},
		{name: "failed", status: models.Failed, expectedResult: false},
		{name: "inactive", status: models.Inactive, expectedResult: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				status := &models.InstanceStatus{Retrieving: models.Retrieving{Status: tc.status}}
				_ = json.NewEncoder(w).Encode(status)
			}))
			defer server.Close()

			client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
			require.NoError(t, err)

			inProgress, err := client.IsRefreshInProgress(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tc.expectedResult, inProgress)
		})
	}
}

func TestDBLabClientTriggerFullRefresh(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/full-refresh", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"OK","message":"Full refresh started"}`))
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.TriggerFullRefresh(context.Background())
		assert.NoError(t, err)
	})
}

func TestDBLabClientUpdateSourceConfig(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		var receivedConfig models.ConfigProjection

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/admin/config", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			err := json.NewDecoder(r.Body).Decode(&receivedConfig)
			require.NoError(t, err)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.UpdateSourceConfig(context.Background(), "clone-host.rds.amazonaws.com", 5432, "postgres", "dbuser", "dbpass")
		require.NoError(t, err)

		assert.Equal(t, "clone-host.rds.amazonaws.com", *receivedConfig.Host)
		assert.Equal(t, int64(5432), *receivedConfig.Port)
		assert.Equal(t, "postgres", *receivedConfig.DBName)
		assert.Equal(t, "dbuser", *receivedConfig.Username)
		assert.Equal(t, "dbpass", *receivedConfig.Password)
	})

	t.Run("error on non-2xx status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"INVALID_CONFIG","message":"invalid configuration"}`))
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.UpdateSourceConfig(context.Background(), "host", 5432, "db", "user", "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid configuration")
	})

	t.Run("error on server error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"INTERNAL_ERROR","message":"internal server error"}`))
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.UpdateSourceConfig(context.Background(), "host", 5432, "db", "user", "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")
	})
}

func TestDBLabClientWaitForRefreshComplete(t *testing.T) {
	t.Run("immediate finish", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			var status models.RetrievalStatus

			if callCount == 1 {
				status = models.Refreshing
			} else {
				status = models.Finished
			}

			resp := &models.InstanceStatus{Retrieving: models.Retrieving{Status: status}}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.WaitForRefreshComplete(context.Background(), 100*time.Millisecond, 5*time.Second)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, callCount, 2)
	})

	t.Run("waits for refresh to start", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			var status models.RetrievalStatus

			switch callCount {
			case 1:
				status = models.Finished
			case 2:
				status = models.Refreshing
			default:
				status = models.Finished
			}

			resp := &models.InstanceStatus{Retrieving: models.Retrieving{Status: status}}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.WaitForRefreshComplete(context.Background(), 100*time.Millisecond, 5*time.Second)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, callCount, 3)
	})

	t.Run("handles failed status", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			var status models.RetrievalStatus

			if callCount == 1 {
				status = models.Refreshing
			} else {
				status = models.Failed
			}

			resp := &models.InstanceStatus{
				Retrieving: models.Retrieving{
					Status: status,
					Alerts: map[models.AlertType]models.Alert{
						models.RefreshFailed: {Message: "test error"},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.WaitForRefreshComplete(context.Background(), 100*time.Millisecond, 5*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "refresh failed")
		assert.Contains(t, err.Error(), "test error")
	})

	t.Run("handles inactive status after start", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			var status models.RetrievalStatus

			if callCount == 1 {
				status = models.Refreshing
			} else {
				status = models.Inactive
			}

			resp := &models.InstanceStatus{Retrieving: models.Retrieving{Status: status}}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.WaitForRefreshComplete(context.Background(), 100*time.Millisecond, 5*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "refresh stopped unexpectedly")
	})

	t.Run("timeout before start", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := &models.InstanceStatus{Retrieving: models.Retrieving{Status: models.Finished}}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.WaitForRefreshComplete(context.Background(), 50*time.Millisecond, 200*time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for refresh to start")
	})

	t.Run("timeout during refresh", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := &models.InstanceStatus{Retrieving: models.Retrieving{Status: models.Refreshing}}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		err = client.WaitForRefreshComplete(context.Background(), 50*time.Millisecond, 200*time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for refresh to complete")
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := &models.InstanceStatus{Retrieving: models.Retrieving{Status: models.Refreshing}}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client, err := NewDBLabClient(&DBLabConfig{APIEndpoint: server.URL, Token: "test-token"})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = client.WaitForRefreshComplete(ctx, 100*time.Millisecond, 5*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

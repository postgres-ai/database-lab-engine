/*
2026 © Postgres.ai
*/

package teleport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestCloneServiceName(t *testing.T) {
	longClone := strings.Repeat("x", 300)
	longFull := "dblab-clone-e-" + longClone + "-1234"
	hash := shortHash(longFull)
	// prefix(14) + available(172) + hashPart(9) + portSuffix(5) = 200
	longWant := "dblab-clone-e-" + strings.Repeat("x", 172) + "-" + hash + "-1234"

	tests := []struct {
		envID   string
		cloneID string
		port    int
		want    string
	}{
		{"prod", "abc123", 5432, "dblab-clone-prod-abc123-5432"},
		{"staging", "my-clone", 6000, "dblab-clone-staging-my-clone-6000"},
		{"e", longClone, 1234, longWant},
	}

	for _, tc := range tests {
		name := CloneServiceName(tc.envID, tc.cloneID, tc.port)
		assert.LessOrEqual(t, len(name), maxNameLen)
		assert.Equal(t, tc.want, name)
		assert.True(t, strings.HasSuffix(name, fmt.Sprintf("-%d", tc.port)),
			"name %q must end with port suffix", name)
	}
}

func TestAPIServiceName(t *testing.T) {
	longEnv := strings.Repeat("x", 300)
	longFull := "dblab-api-" + longEnv
	hash := shortHash(longFull)
	longWant := longFull[:maxNameLen-hashSuffixLen-1] + "-" + hash

	tests := []struct {
		envID string
		want  string
	}{
		{"prod", "dblab-api-prod"},
		{"staging", "dblab-api-staging"},
		{longEnv, longWant},
	}

	for _, tc := range tests {
		name := APIServiceName(tc.envID)
		assert.LessOrEqual(t, len(name), maxNameLen)
		assert.Equal(t, tc.want, name)
	}
}

func TestWebhookPayloadParsing(t *testing.T) {
	raw := `{"event_type":"clone_create","entity_id":"clone-1","host":"localhost","port":5432,"username":"user1","dbname":"postgres"}`

	var p WebhookPayload
	require.NoError(t, json.Unmarshal([]byte(raw), &p))
	assert.Equal(t, "clone_create", p.EventType)
	assert.Equal(t, "clone-1", p.EntityID)
	assert.Equal(t, uint(5432), p.Port)
	assert.Equal(t, "user1", p.Username)
}

func TestWebhookHandler_UnknownEvent(t *testing.T) {
	svc := &service{cfg: &Config{WebhookSecret: "testsecret"}}
	handler := svc.makeWebhookHandler()

	body, _ := json.Marshal(WebhookPayload{EventType: "unknown_event", EntityID: "x"})
	req := httptest.NewRequest(http.MethodPost, "/teleport-sync", bytes.NewReader(body))
	req.Header.Set("DBLab-Webhook-Token", "testsecret")
	rr := httptest.NewRecorder()

	handler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	svc := &service{cfg: &Config{WebhookSecret: "testsecret"}}
	handler := svc.makeWebhookHandler()

	req := httptest.NewRequest(http.MethodPost, "/teleport-sync", strings.NewReader("not-json"))
	req.Header.Set("DBLab-Webhook-Token", "testsecret")
	rr := httptest.NewRecorder()

	handler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestWebhookHandler_WrongMethod(t *testing.T) {
	svc := &service{cfg: &Config{WebhookSecret: "testsecret"}}
	handler := svc.makeWebhookHandler()

	req := httptest.NewRequest(http.MethodGet, "/teleport-sync", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestWebhookHandler_SecretValidation(t *testing.T) {
	svc := &service{cfg: &Config{WebhookSecret: "mysecret"}}
	handler := svc.makeWebhookHandler()

	body, _ := json.Marshal(WebhookPayload{EventType: "clone_create", EntityID: "x", Port: 5432})

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/teleport-sync", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		handler(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("wrong token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/teleport-sync", bytes.NewReader(body))
		req.Header.Set("DBLab-Webhook-Token", "wrongsecret")
		rr := httptest.NewRecorder()
		handler(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("correct token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/teleport-sync", bytes.NewReader(body))
		req.Header.Set("DBLab-Webhook-Token", "mysecret")
		rr := httptest.NewRecorder()
		handler(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestDiffDBs(t *testing.T) {
	cloneA := &models.Clone{ID: "a"}
	cloneB := &models.Clone{ID: "b"}

	t.Run("add and remove", func(t *testing.T) {
		desired := map[string]*models.Clone{"db-a": cloneA, "db-b": cloneB}
		registered := map[string]bool{"db-b": true, "db-c": true}

		toAdd, toRemove := diffDBs(desired, registered)

		require.Len(t, toAdd, 1)
		assert.Equal(t, cloneA, toAdd["db-a"])
		require.Len(t, toRemove, 1)
		assert.True(t, toRemove["db-c"])
	})

	t.Run("both empty", func(t *testing.T) {
		toAdd, toRemove := diffDBs(map[string]*models.Clone{}, map[string]bool{})
		assert.Empty(t, toAdd)
		assert.Empty(t, toRemove)
	})

	t.Run("identical", func(t *testing.T) {
		desired := map[string]*models.Clone{"db-a": cloneA}
		registered := map[string]bool{"db-a": true}

		toAdd, toRemove := diffDBs(desired, registered)
		assert.Empty(t, toAdd)
		assert.Empty(t, toRemove)
	})

	t.Run("empty desired", func(t *testing.T) {
		toAdd, toRemove := diffDBs(map[string]*models.Clone{}, map[string]bool{"db-a": true})
		assert.Empty(t, toAdd)
		require.Len(t, toRemove, 1)
		assert.True(t, toRemove["db-a"])
	})

	t.Run("empty registered", func(t *testing.T) {
		desired := map[string]*models.Clone{"db-a": cloneA}
		toAdd, toRemove := diffDBs(desired, map[string]bool{})
		require.Len(t, toAdd, 1)
		assert.Equal(t, cloneA, toAdd["db-a"])
		assert.Empty(t, toRemove)
	})
}

func TestBuildDesiredDBs(t *testing.T) {
	clones := []*models.Clone{
		{ID: "clone-1", DB: models.Database{Port: "5432", Username: "user1"}},
		{ID: "clone-2", DB: models.Database{Port: "invalid"}},
		{ID: "clone-3", DB: models.Database{Port: "0"}},
	}

	desired := buildDesiredDBs(clones, "prod")

	require.Len(t, desired, 1)

	key := "dblab-clone-prod-clone-1-5432"
	require.Contains(t, desired, key)
	assert.Equal(t, "clone-1", desired[key].ID)
	assert.Equal(t, "5432", desired[key].DB.Port)
	assert.Equal(t, "user1", desired[key].DB.Username)
}

func TestSanitizeYAMLValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid alphanumeric", "clone-abc-123", false},
		{"valid with dots", "localhost:5432", false},
		{"valid URL-like", "http://localhost:2345", false},
		{"valid with slashes", "db/my-clone", false},
		{"valid with underscore", "my_clone", false},
		{"valid with at sign", "user@host", false},
		{"empty string", "", true},
		{"newline injection", "valid\ninjected: true", true},
		{"quote injection", "valid\"injected", true},
		{"space injection", "valid injected", true},
		{"yaml block scalar", "|", true},
		{"yaml brace", "{key: val}", true},
		{"yaml bracket", "[item]", true},
		{"comment injection", "value #comment", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sanitizeYAMLValue(tc.value, "field")
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckEdition(t *testing.T) {
	tests := []struct {
		name      string
		edition   string
		wantErr   bool
		errSubstr string
	}{
		{"standard edition allowed", global.StandardEdition, false, ""},
		{"enterprise edition allowed", global.EnterpriseEdition, false, ""},
		{"community edition blocked", global.CommunityEdition, true, "requires Standard or Enterprise edition"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				status := models.InstanceStatus{
					Engine: models.Engine{Edition: tc.edition},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(status)
			}))
			defer srv.Close()

			cfg := &Config{
				DblabURL:   srv.URL,
				DblabToken: "test-token",
			}

			client, clientErr := newDblabClient(cfg)
			require.NoError(t, clientErr)

			err := checkEdition(client)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckEdition_ConnectionError(t *testing.T) {
	client, err := newDblabClient(&Config{
		DblabURL:   "http://localhost:1",
		DblabToken: "test-token",
	})
	require.NoError(t, err)

	err = checkEdition(client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check DBLab instance status")
}

func TestWebhookHandler_OversizedBody(t *testing.T) {
	svc := &service{cfg: &Config{WebhookSecret: "testsecret"}}
	handler := svc.makeWebhookHandler()

	largeBody := strings.Repeat("x", maxRequestBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/teleport-sync", strings.NewReader(largeBody))
	req.Header.Set("DBLab-Webhook-Token", "testsecret")
	rr := httptest.NewRecorder()

	handler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestParseListDBsOutput(t *testing.T) {
	t.Run("filters by dblab label and instance", func(t *testing.T) {
		data := `[
			{"metadata":{"name":"dblab-clone-prod-a-5432","labels":{"dblab":"true","dblab_instance":"prod"}}},
			{"metadata":{"name":"dblab-clone-staging-b-6000","labels":{"dblab":"true","dblab_instance":"staging"}}},
			{"metadata":{"name":"manual-db","labels":{"dblab_instance":"prod"}}},
			{"metadata":{"name":"other-db","labels":{"dblab":"false","dblab_instance":"prod"}}}
		]`
		result, err := parseListDBsOutput([]byte(data), "prod")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.True(t, result["dblab-clone-prod-a-5432"])
	})

	t.Run("matches legacy resources without instance label", func(t *testing.T) {
		data := `[
			{"metadata":{"name":"legacy-prod","labels":{"dblab":"true","environment":"prod"}}},
			{"metadata":{"name":"legacy-staging","labels":{"dblab":"true","environment":"staging"}}}
		]`
		result, err := parseListDBsOutput([]byte(data), "prod")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.True(t, result["legacy-prod"])
	})

	t.Run("custom environment does not match instance", func(t *testing.T) {
		data := `[{"metadata":{"name":"other-instance","labels":{"dblab":"true","dblab_instance":"ci","environment":"prod"}}}]`
		result, err := parseListDBsOutput([]byte(data), "prod")
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("dblab resource without any ownership label is not matched", func(t *testing.T) {
		// dblab=true but neither dblab_instance nor environment: the legacy
		// fallback must fail (it requires environment == envID).
		data := `[{"metadata":{"name":"orphan","labels":{"dblab":"true"}}}]`
		result, err := parseListDBsOutput([]byte(data), "prod")
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("empty list", func(t *testing.T) {
		result, err := parseListDBsOutput([]byte("[]"), "prod")
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("malformed json", func(t *testing.T) {
		_, err := parseListDBsOutput([]byte("not-json"), "prod")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse tctl output")
	})

	t.Run("missing labels", func(t *testing.T) {
		data := `[{"metadata":{"name":"no-labels","labels":{}}}]`
		result, err := parseListDBsOutput([]byte(data), "prod")
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestBuildDBYAML(t *testing.T) {
	t.Run("with username", func(t *testing.T) {
		res := dbResource{Name: "dblab-clone-prod-abc-5432", Port: 5432, EnvID: "prod", CloneID: "abc", Username: "testuser"}
		yaml, err := buildDBYAML(res, nil)
		require.NoError(t, err)
		s := string(yaml)
		assert.Contains(t, s, `name: "dblab-clone-prod-abc-5432"`)
		assert.Contains(t, s, `dblab: "true"`)
		assert.Contains(t, s, `dblab_instance: "prod"`)
		assert.Contains(t, s, `environment: "prod"`)
		assert.Contains(t, s, `clone_id: "abc"`)
		assert.Contains(t, s, `dblab_user: "testuser"`)
		assert.Contains(t, s, `uri: "127.0.0.1:5432"`)
	})

	t.Run("without username", func(t *testing.T) {
		res := dbResource{Name: "dblab-clone-prod-abc-5432", Port: 5432, EnvID: "prod", CloneID: "abc", Username: ""}
		yaml, err := buildDBYAML(res, nil)
		require.NoError(t, err)
		s := string(yaml)
		assert.NotContains(t, s, "dblab_user")
		assert.Contains(t, s, `clone_id: "abc"`)
	})

	t.Run("custom labels", func(t *testing.T) {
		res := dbResource{Name: "dblab-clone-reg-abc-5432", Port: 5432, EnvID: "gitlab-registry", CloneID: "abc"}
		custom := map[string]string{"environment": "gprd", "db-type": "registry", "service": "dblab"}
		yaml, err := buildDBYAML(res, custom)
		require.NoError(t, err)
		s := string(yaml)
		assert.Contains(t, s, `environment: "gprd"`)
		assert.Contains(t, s, `db-type: "registry"`)
		assert.Contains(t, s, `service: "dblab"`)
		assert.Contains(t, s, `dblab_instance: "gitlab-registry"`)
		assert.NotContains(t, s, `environment: "gitlab-registry"`)
	})

	t.Run("labels rendered in deterministic sorted order", func(t *testing.T) {
		res := dbResource{Name: "dblab-clone-reg-abc-5432", Port: 5432, EnvID: "gitlab-registry", CloneID: "abc"}
		custom := map[string]string{"environment": "gprd", "db-type": "registry", "service": "dblab"}
		yaml, err := buildDBYAML(res, custom)
		require.NoError(t, err)
		s := string(yaml)
		// Keys must be emitted sorted so tctl create stays idempotent (no diff noise).
		assert.Less(t, strings.Index(s, `db-type:`), strings.Index(s, `dblab:`))
		assert.Less(t, strings.Index(s, `dblab:`), strings.Index(s, `dblab_instance:`))
		assert.Less(t, strings.Index(s, `dblab_instance:`), strings.Index(s, `environment:`))
		assert.Less(t, strings.Index(s, `environment:`), strings.Index(s, `service:`))
	})

	t.Run("invalid name", func(t *testing.T) {
		res := dbResource{Name: "bad name with spaces", Port: 5432, EnvID: "prod", CloneID: "abc"}
		_, err := buildDBYAML(res, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid db resource")
	})

	t.Run("invalid clone_id", func(t *testing.T) {
		res := dbResource{Name: "valid-name", Port: 5432, EnvID: "prod", CloneID: "bad\ninjection"}
		_, err := buildDBYAML(res, nil)
		require.Error(t, err)
	})

	t.Run("invalid username", func(t *testing.T) {
		res := dbResource{Name: "valid-name", Port: 5432, EnvID: "prod", CloneID: "abc", Username: "user\ninjection"}
		_, err := buildDBYAML(res, nil)
		require.Error(t, err)
	})

	t.Run("invalid custom label value", func(t *testing.T) {
		res := dbResource{Name: "valid-name", Port: 5432, EnvID: "prod", CloneID: "abc"}
		_, err := buildDBYAML(res, map[string]string{"service": "bad\ninjection"})
		require.Error(t, err)
	})
}

func TestBuildAppYAML(t *testing.T) {
	t.Run("default labels", func(t *testing.T) {
		yaml, err := buildAppYAML("dblab-app-prod", "https://127.0.0.1:2346", "prod", nil)
		require.NoError(t, err)
		s := string(yaml)
		assert.Contains(t, s, `name: "dblab-app-prod"`)
		assert.Contains(t, s, `dblab: "true"`)
		assert.Contains(t, s, `dblab_instance: "prod"`)
		assert.Contains(t, s, `environment: "prod"`)
		assert.Contains(t, s, `uri: "https://127.0.0.1:2346"`)
		// App resources are not clone-scoped: no DB-only labels.
		assert.NotContains(t, s, "clone_id")
		assert.NotContains(t, s, "dblab_user")
	})

	t.Run("custom labels suppress default environment", func(t *testing.T) {
		custom := map[string]string{"environment": "gprd", "db-type": "registry", "service": "dblab"}
		yaml, err := buildAppYAML("dblab-app-reg", "https://127.0.0.1:2346", "gitlab-registry", custom)
		require.NoError(t, err)
		s := string(yaml)
		assert.Contains(t, s, `environment: "gprd"`)
		assert.Contains(t, s, `db-type: "registry"`)
		assert.Contains(t, s, `service: "dblab"`)
		assert.Contains(t, s, `dblab_instance: "gitlab-registry"`)
		assert.NotContains(t, s, `environment: "gitlab-registry"`)
	})

	t.Run("invalid name", func(t *testing.T) {
		_, err := buildAppYAML("bad name with spaces", "https://127.0.0.1:2346", "prod", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid app resource")
	})

	t.Run("invalid custom label value", func(t *testing.T) {
		_, err := buildAppYAML("dblab-app-prod", "https://127.0.0.1:2346", "prod", map[string]string{"service": "bad\ninjection"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid app resource")
	})
}

func TestParseLabels(t *testing.T) {
	t.Run("valid pairs", func(t *testing.T) {
		labels, err := parseLabels([]string{"environment=gprd", "db-type=registry", "service=dblab"})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"environment": "gprd", "db-type": "registry", "service": "dblab"}, labels)
	})

	t.Run("empty input", func(t *testing.T) {
		labels, err := parseLabels(nil)
		require.NoError(t, err)
		assert.Empty(t, labels)
	})

	t.Run("missing separator", func(t *testing.T) {
		_, err := parseLabels([]string{"noseparator"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "key=value")
	})

	t.Run("empty key", func(t *testing.T) {
		_, err := parseLabels([]string{"=value"})
		require.Error(t, err)
	})

	t.Run("empty value", func(t *testing.T) {
		_, err := parseLabels([]string{"service="})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "key=value")
	})

	t.Run("reserved key", func(t *testing.T) {
		_, err := parseLabels([]string{"dblab_instance=foo"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reserved")
	})

	t.Run("duplicate key rejected", func(t *testing.T) {
		_, err := parseLabels([]string{"service=a", "service=b"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("invalid key characters", func(t *testing.T) {
		_, err := parseLabels([]string{"db type=registry"})
		require.Error(t, err)
	})

	t.Run("invalid value characters", func(t *testing.T) {
		_, err := parseLabels([]string{"service=has space"})
		require.Error(t, err)
	})
}

func TestBaseLabels(t *testing.T) {
	t.Run("defaults environment to instance id", func(t *testing.T) {
		labels := baseLabels("prod", nil)
		assert.Equal(t, "true", labels["dblab"])
		assert.Equal(t, "prod", labels["dblab_instance"])
		assert.Equal(t, "prod", labels["environment"])
		assert.NotContains(t, labels, "clone_id")
	})

	t.Run("custom environment suppresses default and reserved labels win", func(t *testing.T) {
		labels := baseLabels("gitlab-registry", map[string]string{"environment": "gprd", "service": "dblab"})
		assert.Equal(t, "gprd", labels["environment"])
		assert.Equal(t, "dblab", labels["service"])
		assert.Equal(t, "gitlab-registry", labels["dblab_instance"])
		assert.Equal(t, "true", labels["dblab"])
	})
}

func TestWebhookHandler_CloneCreateValidation(t *testing.T) {
	svc := &service{cfg: &Config{WebhookSecret: "secret"}}
	handler := svc.makeWebhookHandler()

	t.Run("missing entity_id", func(t *testing.T) {
		body, _ := json.Marshal(WebhookPayload{EventType: "clone_create", EntityID: "", Port: 5432})
		req := httptest.NewRequest(http.MethodPost, "/teleport-sync", bytes.NewReader(body))
		req.Header.Set("DBLab-Webhook-Token", "secret")
		rr := httptest.NewRecorder()
		handler(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("missing port", func(t *testing.T) {
		body, _ := json.Marshal(WebhookPayload{EventType: "clone_create", EntityID: "clone-1", Port: 0})
		req := httptest.NewRequest(http.MethodPost, "/teleport-sync", bytes.NewReader(body))
		req.Header.Set("DBLab-Webhook-Token", "secret")
		rr := httptest.NewRecorder()
		handler(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestWebhookHandler_CloneDeleteValidation(t *testing.T) {
	svc := &service{cfg: &Config{WebhookSecret: "secret"}}
	handler := svc.makeWebhookHandler()

	body, _ := json.Marshal(WebhookPayload{EventType: "clone_delete", EntityID: "", Port: 5432})
	req := httptest.NewRequest(http.MethodPost, "/teleport-sync", bytes.NewReader(body))
	req.Header.Set("DBLab-Webhook-Token", "secret")
	rr := httptest.NewRecorder()

	handler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

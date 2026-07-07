package srv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/projection"
	yamlUtils "gitlab.com/postgres-ai/database-lab/v3/pkg/util/yaml"
)

func ptrString(s string) *string { return &s }
func ptrBool(b bool) *bool       { return &b }
func ptrInt64(i int64) *int64    { return &i }

func mustMarshal(t *testing.T, v interface{}) string {
	t.Helper()

	b, err := json.Marshal(v)
	require.NoError(t, err)

	return string(b)
}

func newProbeTestServer(t *testing.T, disableConfigMod bool) *Server {
	t.Helper()

	pl, err := platform.New(context.Background(), platform.Config{}, "instanceID")
	require.NoError(t, err)

	return &Server{
		Config:   &config.Config{DisableConfigModification: disableConfigMod},
		Platform: pl,
		tm:       telemetry.New(pl, "instanceID"),
	}
}

func TestProbeSource_DisableConfigModification(t *testing.T) {
	srv := newProbeTestServer(t, true)

	req := httptest.NewRequest(http.MethodPost, "/admin/probe-source", strings.NewReader(`{"url":"postgres://x@y/z","password":"p"}`))
	rec := httptest.NewRecorder()

	srv.probeSource(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "disabled by admin")
}

func TestProbeSource_InvalidJSON(t *testing.T) {
	srv := newProbeTestServer(t, false)

	req := httptest.NewRequest(http.MethodPost, "/admin/probe-source", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()

	srv.probeSource(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTestDBSource_PhysicalModeRejected(t *testing.T) {
	srv := newProbeTestServer(t, false)
	srv.Retrieval = &retrieval.Retrieval{State: retrieval.State{Mode: models.Physical}}

	body := `{"host":"db.example.com","port":"5432","dbname":"shop","username":"app","password":"p"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/test-db-source", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.testDBSource(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "the endpoint is only available in the Logical mode of the data retrieval")
}

func TestProbeSource_PasswordInURL(t *testing.T) {
	srv := newProbeTestServer(t, false)

	body := `{"url":"postgres://alice:secret@db.example.com/shop","password":"separate"}`

	req := httptest.NewRequest(http.MethodPost, "/admin/probe-source", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.probeSource(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "password must not be embedded")
	require.NotContains(t, rec.Body.String(), "secret", "the embedded password must never appear in the error response")
}

func TestConfigProbedPayload_ContainsOnlyProviderKey(t *testing.T) {
	// guard against accidentally widening the telemetry payload to include
	// URL, host, dbname, or username — those are sensitive.
	payload := telemetry.ConfigProbed{Provider: "rds"}

	require.Equal(t, "rds", payload.Provider)
	require.JSONEq(t, `{"provider":"rds"}`, mustMarshal(t, payload))
}

func TestRequestedRetrievalMode(t *testing.T) {
	testCases := []struct {
		name     string
		obj      map[string]interface{}
		fallback models.RetrievalMode
		want     models.RetrievalMode
	}{
		{name: "explicit logical wins over fallback", obj: map[string]interface{}{"retrievalMode": "logical"}, fallback: models.Physical, want: models.Logical},
		{name: "explicit physical wins over fallback", obj: map[string]interface{}{"retrievalMode": "physical"}, fallback: models.Logical, want: models.Physical},
		{name: "missing field falls back to running mode", obj: map[string]interface{}{}, fallback: models.Logical, want: models.Logical},
		{name: "empty string falls back", obj: map[string]interface{}{"retrievalMode": ""}, fallback: models.Physical, want: models.Physical},
		{name: "non-string value falls back", obj: map[string]interface{}{"retrievalMode": 42}, fallback: models.Logical, want: models.Logical},
		{name: "unknown string is passed through", obj: map[string]interface{}{"retrievalMode": "garbage"}, fallback: models.Logical, want: models.RetrievalMode("garbage")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, requestedRetrievalMode(tc.obj, tc.fallback))
		})
	}
}

func TestGuardModeFields(t *testing.T) {
	t.Run("logical mode accepts logical-only fields", func(t *testing.T) {
		proj := &models.ConfigProjection{Host: ptrString("db.example.com"), Username: ptrString("alice")}
		require.NoError(t, guardModeFields(models.Logical, proj))
	})

	t.Run("logical mode rejects physical field", func(t *testing.T) {
		proj := &models.ConfigProjection{Host: ptrString("db.example.com"), PhysicalTool: ptrString("walg")}
		err := guardModeFields(models.Logical, proj)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "physicalTool")
	})

	t.Run("physical mode accepts physical-only fields", func(t *testing.T) {
		proj := &models.ConfigProjection{PhysicalTool: ptrString("walg"), PhysicalWalgBackupName: ptrString("LATEST")}
		require.NoError(t, guardModeFields(models.Physical, proj))
	})

	t.Run("physical mode rejects logical field", func(t *testing.T) {
		proj := &models.ConfigProjection{PhysicalTool: ptrString("walg"), Host: ptrString("db.example.com")}
		err := guardModeFields(models.Physical, proj)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host")
	})

	t.Run("physical mode rejects rdsIam field", func(t *testing.T) {
		proj := &models.ConfigProjection{PhysicalTool: ptrString("walg"), RDSIAMDBInstance: ptrString("my-db")}
		err := guardModeFields(models.Physical, proj)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rdsIamDbInstanceIdentifier")
	})

	t.Run("logical mode rejects PhysicalEnvs map", func(t *testing.T) {
		proj := &models.ConfigProjection{Host: ptrString("db.example.com"), PhysicalEnvs: map[string]interface{}{"K": "V"}}
		err := guardModeFields(models.Logical, proj)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "physicalEnvs")
	})

	t.Run("shared fields are accepted in either mode", func(t *testing.T) {
		shared := &models.ConfigProjection{Debug: ptrBool(true), DockerImage: ptrString("postgresai/extended-postgres:18")}
		require.NoError(t, guardModeFields(models.Logical, shared))
		require.NoError(t, guardModeFields(models.Physical, shared))
	})

	t.Run("unknown mode does not gate", func(t *testing.T) {
		proj := &models.ConfigProjection{Host: ptrString("h"), PhysicalTool: ptrString("walg")}
		require.NoError(t, guardModeFields(models.Unknown, proj),
			"the dispatcher rejects Unknown before guardModeFields runs; the helper itself should not")
	})

	t.Run("logical mode accepts port and parallelism", func(t *testing.T) {
		proj := &models.ConfigProjection{Port: ptrInt64(5432), DumpParallelJobs: ptrInt64(4)}
		require.NoError(t, guardModeFields(models.Logical, proj))
	})

	t.Run("logical mode accepts connectionString", func(t *testing.T) {
		proj := &models.ConfigProjection{ConnectionString: ptrString("postgres://alice@db.example.com/shop")}
		require.NoError(t, guardModeFields(models.Logical, proj))
	})

	t.Run("physical mode rejects connectionString", func(t *testing.T) {
		proj := &models.ConfigProjection{PhysicalTool: ptrString("walg"), ConnectionString: ptrString("postgres://alice@db.example.com/shop")}
		err := guardModeFields(models.Physical, proj)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "connectionString")
	})
}

func TestProjectionRoundTrip_ConnectionString(t *testing.T) {
	const base = `
retrieval:
  spec:
    logicalDump:
      options:
        source:
          connection:
            host: old-host
`
	const connStr = "postgres://alice@db.example.com:5433/shop?sslmode=require"

	node := &yaml.Node{}
	require.NoError(t, yaml.Unmarshal([]byte(base), node))

	store := &models.ConfigProjection{ConnectionString: ptrString(connStr)}
	require.NoError(t, projection.StoreYaml(store, node, projection.StoreOptions{Groups: []string{"default", "sensitive"}}))

	leaf, ok := yamlUtils.FindNodeAtPath(node,
		[]string{"retrieval", "spec", "logicalDump", "options", "source", "connectionString"})
	require.True(t, ok, "connectionString must persist under source")
	assert.Equal(t, connStr, leaf.Value)

	reloaded := &models.ConfigProjection{}
	require.NoError(t, projection.LoadYaml(reloaded, node, projection.LoadOptions{Groups: []string{"default", "sensitive"}}))
	require.NotNil(t, reloaded.ConnectionString)
	assert.Equal(t, connStr, *reloaded.ConnectionString)

	// connectionString is in the "sensitive" group, so the GET path (default
	// group only) must not surface it — mirroring how the discrete password is
	// withheld from GET /admin/config.
	echoed := &models.ConfigProjection{}
	require.NoError(t, projection.LoadYaml(echoed, node, projection.LoadOptions{Groups: []string{"default"}}))
	require.Nil(t, echoed.ConnectionString, "connectionString must not be echoed on the default (GET) group")

	require.Error(t, guardModeFields(models.Physical, reloaded),
		"connectionString is logical-only and must be rejected under physical mode")
}

func TestLoadJSON_RequestReadsConnectionString(t *testing.T) {
	// mirror the nested JSON a client (CLI/UI) posts to /admin/config; the apply
	// handler runs projection.LoadJSON over exactly this shape, so this proves the
	// request side decodes the new field, not just the YAML store/load round-trip.
	const connStr = "postgres://alice@db.example.com:5432/shop?sslmode=require"

	objMap := map[string]interface{}{
		"retrievalMode": "logical",
		"retrieval": map[string]interface{}{
			"spec": map[string]interface{}{
				"logicalDump": map[string]interface{}{
					"options": map[string]interface{}{
						"source": map[string]interface{}{
							"connectionString": connStr,
							"connection":       map[string]interface{}{"host": "db.example.com"},
						},
					},
				},
			},
		},
	}

	// the apply handler loads with both groups so the sensitive connectionString
	// is decoded from the request body.
	proj := &models.ConfigProjection{}
	require.NoError(t, projection.LoadJSON(proj, objMap, projection.LoadOptions{Groups: []string{"default", "sensitive"}}))
	require.NotNil(t, proj.ConnectionString)
	assert.Equal(t, connStr, *proj.ConnectionString)
}

func TestValidateSourceConnectionString(t *testing.T) {
	testCases := []struct {
		name    string
		connStr *string
		wantErr string
	}{
		{name: "nil is a no-op", connStr: nil, wantErr: ""},
		{name: "empty string is a no-op (clear)", connStr: ptrString(""), wantErr: ""},
		{name: "password-less uri accepted", connStr: ptrString("postgres://alice@db.example.com:5432/shop?sslmode=require"), wantErr: ""},
		{name: "password-less dsn accepted", connStr: ptrString("host=db.example.com user=alice dbname=shop"), wantErr: ""},
		{name: "uri embedded password rejected", connStr: ptrString("postgres://alice:secret@db.example.com/shop"), wantErr: "password must not be embedded"},
		{name: "dsn embedded password rejected", connStr: ptrString("host=db.example.com user=alice password=secret dbname=shop"), wantErr: "password must not be embedded"},
		{name: "multi-host rejected", connStr: ptrString("postgres://alice@db1.example.com,db2.example.com/shop"), wantErr: "multi-host"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSourceConnectionString(tc.connStr)

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)

			if tc.connStr != nil {
				assert.NotContains(t, err.Error(), "secret", "the embedded password must never appear in the rejection message")
			}
		})
	}
}

func TestSetProjectedAdminConfig_RejectsEmbeddedPasswordConnectionString(t *testing.T) {
	srv := newProbeTestServer(t, false)
	srv.Retrieval = &retrieval.Retrieval{State: retrieval.State{Mode: models.Logical}}

	body := `{
		"retrievalMode": "logical",
		"retrieval": {"spec": {"logicalDump": {"options": {"source": {
			"connectionString": "postgres://alice:secret@db.example.com/shop"
		}}}}}
	}`

	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.setProjectedAdminConfig(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "password must not be embedded")
	require.NotContains(t, rec.Body.String(), "secret", "the embedded password must never appear in the error response")
}

func TestEnsureLogicalPipeline(t *testing.T) {
	const specs = `
  spec:
    logicalDump:
      options: {}
    logicalRestore:
      options: {}
    logicalSnapshot:
      options: {}
`

	jobsAfter := func(t *testing.T, doc string) []string {
		t.Helper()

		node := &yaml.Node{}
		require.NoError(t, yaml.Unmarshal([]byte(doc), node))
		require.NoError(t, ensureLogicalPipeline(node))

		jobsNode, ok := yamlUtils.FindNodeAtPath(node, []string{"retrieval", "jobs"})
		require.True(t, ok)

		jobs := make([]string, 0, len(jobsNode.Content))
		for _, item := range jobsNode.Content {
			jobs = append(jobs, item.Value)
		}

		return jobs
	}

	t.Run("adds missing logicalRestore to complete the pipeline", func(t *testing.T) {
		jobs := jobsAfter(t, "retrieval:\n  jobs:\n    - logicalDump\n    - logicalSnapshot\n"+specs)
		assert.Equal(t, []string{"logicalDump", "logicalRestore", "logicalSnapshot"}, jobs)
	})

	t.Run("idempotent when pipeline already complete", func(t *testing.T) {
		jobs := jobsAfter(t, "retrieval:\n  jobs:\n    - logicalDump\n    - logicalRestore\n    - logicalSnapshot\n"+specs)
		assert.Equal(t, []string{"logicalDump", "logicalRestore", "logicalSnapshot"}, jobs)
	})

	t.Run("builds full pipeline from empty jobs list", func(t *testing.T) {
		jobs := jobsAfter(t, "retrieval:\n  jobs:\n"+specs)
		assert.Equal(t, []string{"logicalDump", "logicalRestore", "logicalSnapshot"}, jobs)
	})

	t.Run("omits logicalRestore when immediateRestore is enabled", func(t *testing.T) {
		doc := "retrieval:\n  jobs:\n    - logicalDump\n" +
			"  spec:\n    logicalDump:\n      options:\n        immediateRestore:\n          enabled: true\n" +
			"    logicalRestore:\n      options: {}\n    logicalSnapshot:\n      options: {}\n"
		jobs := jobsAfter(t, doc)
		assert.Equal(t, []string{"logicalDump", "logicalSnapshot"}, jobs)
	})

	t.Run("omits logicalRestore for non-canonical immediateRestore bool", func(t *testing.T) {
		doc := "retrieval:\n  jobs:\n    - logicalDump\n" +
			"  spec:\n    logicalDump:\n      options:\n        immediateRestore:\n          enabled: \"yes\"\n" +
			"    logicalRestore:\n      options: {}\n    logicalSnapshot:\n      options: {}\n"
		jobs := jobsAfter(t, doc)
		assert.Equal(t, []string{"logicalDump", "logicalSnapshot"}, jobs)
	})

	t.Run("skips jobs whose spec is absent", func(t *testing.T) {
		doc := "retrieval:\n  jobs:\n    - logicalDump\n" +
			"  spec:\n    logicalDump:\n      options: {}\n    logicalSnapshot:\n      options: {}\n"
		jobs := jobsAfter(t, doc)
		assert.Equal(t, []string{"logicalDump", "logicalSnapshot"}, jobs)
	})

	t.Run("errors when retrieval section is absent", func(t *testing.T) {
		node := &yaml.Node{}
		require.NoError(t, yaml.Unmarshal([]byte("global:\n  debug: true\n"), node))
		assert.Error(t, ensureLogicalPipeline(node))
	})
}

func TestCustomOptions(t *testing.T) {
	testCases := []struct {
		customOptions  []interface{}
		expectedResult error
	}{
		{
			customOptions:  []interface{}{"--verbose"},
			expectedResult: nil,
		},
		{
			customOptions:  []interface{}{"--exclude-scheme=test_scheme"},
			expectedResult: nil,
		},
		{
			customOptions:  []interface{}{`--exclude-scheme="test_scheme"`},
			expectedResult: nil,
		},
		{
			customOptions:  []interface{}{"--table=$(echo 'test')"},
			expectedResult: errInvalidOption,
		},
		{
			customOptions:  []interface{}{"--table=test&table"},
			expectedResult: errInvalidOption,
		},
		{
			customOptions:  []interface{}{5},
			expectedResult: errInvalidOptionType,
		},
	}

	for _, tc := range testCases {
		validationResult := validateCustomOptions(tc.customOptions)

		require.ErrorIs(t, validationResult, tc.expectedResult)
	}
}

func TestValidateCustomOptions_AdditionalCases(t *testing.T) {
	t.Run("empty options list is valid", func(t *testing.T) {
		assert.NoError(t, validateCustomOptions([]interface{}{}))
	})

	t.Run("nil options list is valid", func(t *testing.T) {
		assert.NoError(t, validateCustomOptions(nil))
	})

	t.Run("multiple valid options", func(t *testing.T) {
		opts := []interface{}{"--verbose", "--jobs=4", "--format=custom"}
		assert.NoError(t, validateCustomOptions(opts))
	})

	t.Run("first valid second invalid stops with error", func(t *testing.T) {
		opts := []interface{}{"--verbose", "--table=$(injection)"}
		err := validateCustomOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidOption)
	})

	t.Run("option with spaces is rejected", func(t *testing.T) {
		opts := []interface{}{"--table=test table"}
		err := validateCustomOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidOption)
	})

	t.Run("option with semicolon is rejected", func(t *testing.T) {
		opts := []interface{}{"--table=test;drop"}
		err := validateCustomOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidOption)
	})

	t.Run("option with pipe is rejected", func(t *testing.T) {
		opts := []interface{}{"--table=test|cat"}
		err := validateCustomOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidOption)
	})

	t.Run("non-string types are rejected", func(t *testing.T) {
		testCases := []struct {
			name string
			opt  interface{}
		}{
			{name: "float", opt: 3.14},
			{name: "bool", opt: true},
			{name: "slice", opt: []string{"a"}},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := validateCustomOptions([]interface{}{tc.opt})
				assert.ErrorIs(t, err, errInvalidOptionType)
			})
		}
	})

	t.Run("valid option characters", func(t *testing.T) {
		validOpts := []interface{}{
			"simple",
			"With-Hyphens",
			"under_scores",
			"equal=sign",
			`with"quotes"`,
			"MixedCase123",
		}
		for _, opt := range validOpts {
			assert.NoError(t, validateCustomOptions([]interface{}{opt}), "option %q should be valid", opt)
		}
	})
}

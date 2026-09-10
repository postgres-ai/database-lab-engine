/*
2026 © Postgres.ai
*/

package localinstall

import (
	"encoding/json"
	"flag"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const preloadLibs = "pg_stat_statements"

func sampleProposal() *models.ProposedConfig {
	return &models.ProposedConfig{
		Source:                 models.SourceConnection{Host: "db.example.com", Port: 5432, Username: "alice", DBName: "shop"},
		DetectedProvider:       "rds",
		DockerImage:            "rds",
		DockerTag:              "16-0.8.0",
		ResolvedImage:          "registry.gitlab.com/postgres-ai/se-images/rds:16-0.8.0",
		PgMajorVersion:         16,
		CollationVersion:       "2.36",
		Databases:              []string{"shop"},
		SharedBuffers:          "4GB",
		SharedPreloadLibraries: preloadLibs,
		QueryTuning:            map[string]string{"work_mem": "64MB", "random_page_cost": "1.1"},
	}
}

func TestValidate(t *testing.T) {
	require.NoError(t, installOptions{sourceURL: "postgres://x@y/z"}.validate())
	require.Error(t, installOptions{}.validate())
	require.Error(t, installOptions{sourceURL: "postgres://x@y/z", start: true, noStart: true}.validate())
}

func TestShouldStartRefresh(t *testing.T) {
	tests := []struct {
		name    string
		status  models.RetrievalStatus
		start   bool
		noStart bool
		want    bool
	}{
		{name: "no-start always wins", status: models.Finished, start: true, noStart: true, want: false},
		{name: "pending defers to the auto refresh", status: models.Pending, start: true, want: false},
		{name: "refreshing skips a second trigger", status: models.Refreshing, start: true, want: false},
		{name: "snapshotting skips a second trigger", status: models.Snapshotting, start: true, want: false},
		{name: "start on a settled instance", status: models.Finished, start: true, want: true},
		{name: "default does nothing when settled", status: models.Finished, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, shouldStartRefresh(tc.status, tc.start, tc.noStart))
		})
	}
}

func TestRenderPreview(t *testing.T) {
	out := renderPreview(sampleProposal(), installOptions{})

	assert.Contains(t, out, "alice@db.example.com:5432/shop")
	assert.Contains(t, out, "rds")
	assert.Contains(t, out, "16")
	assert.Contains(t, out, "2.36")
	assert.Contains(t, out, "registry.gitlab.com/postgres-ai/se-images/rds:16-0.8.0")
	assert.Contains(t, out, "4GB")
	assert.Contains(t, out, preloadLibs)
	assert.Contains(t, out, "random_page_cost: 1.1")
	assert.Contains(t, out, "work_mem: 64MB")
	assert.NotContains(t, out, "secret", "the preview must never carry a credential")
}

func TestRenderPreview_ShowsOverriddenValues(t *testing.T) {
	opts := installOptions{sharedBuffers: "8GB", sharedPreloadLibraries: "pg_stat_statements,auto_explain"}
	out := renderPreview(sampleProposal(), opts)

	assert.Contains(t, out, "shared_buffers: 8GB")
	assert.Contains(t, out, "Preload libs:   pg_stat_statements,auto_explain")
	assert.NotContains(t, out, "shared_buffers: 4GB", "the preview must not show a value the apply step overrides")
	assert.Contains(t, out, "Query tuning:")
	assert.Contains(t, out, "work_mem: 64MB")
}

func TestRenderPreview_NoQueryTuningSection(t *testing.T) {
	p := &models.ProposedConfig{SharedBuffers: "4GB", SharedPreloadLibraries: preloadLibs}
	p.QueryTuning = map[string]string{sharedBuffersKey: "1GB", sharedPreloadLibrariesKey: "x"}

	out := renderPreview(p, installOptions{})

	assert.NotContains(t, out, "Query tuning:", "reserved keys have their own lines and must not open the section")
	assert.Contains(t, out, "shared_buffers: 4GB")
}

func TestRenderPreview_GlibcWarning(t *testing.T) {
	t.Run("warns when PG15+ libc source lacks a glibc-pinned tag", func(t *testing.T) {
		out := renderPreview(sampleProposal(), installOptions{})
		assert.Contains(t, out, "WARNING")
		assert.Contains(t, out, "2.36")
	})

	t.Run("no warning when the tag is glibc-pinned", func(t *testing.T) {
		p := sampleProposal()
		p.DockerTag = "16-0.8.0-glibc236"
		assert.NotContains(t, renderPreview(p, installOptions{}), "WARNING")
	})

	t.Run("no warning below PG15", func(t *testing.T) {
		p := sampleProposal()
		p.PgMajorVersion = 14
		assert.NotContains(t, renderPreview(p, installOptions{}), "WARNING")
	})

	t.Run("no warning without a collation version", func(t *testing.T) {
		p := sampleProposal()
		p.CollationVersion = ""
		assert.NotContains(t, renderPreview(p, installOptions{}), "WARNING")
	})
}

func projectionMap(t *testing.T, raw json.RawMessage) map[string]interface{} {
	t.Helper()

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &m))

	return m
}

func dig(t *testing.T, m map[string]interface{}, path ...string) interface{} {
	t.Helper()

	cur := interface{}(m)
	for _, key := range path {
		asMap, ok := cur.(map[string]interface{})
		require.Truef(t, ok, "expected a map at %q", key)
		cur, ok = asMap[key]
		require.Truef(t, ok, "missing key %q", key)
	}

	return cur
}

func TestBuildProjection_DiscreteFields(t *testing.T) {
	raw, err := buildProjection(sampleProposal(), installOptions{
		sourceURL: "postgres://alice@db.example.com:5432/shop", password: "secret",
	})
	require.NoError(t, err)

	m := projectionMap(t, raw)
	assert.Equal(t, "logical", dig(t, m, "retrievalMode"))

	source := dig(t, m, "retrieval", "spec", "logicalDump", "options", "source").(map[string]interface{})
	_, hasConnStr := source["connectionString"]
	assert.False(t, hasConnStr, "no extra params → discrete fields, no connectionString")

	conn := dig(t, m, "retrieval", "spec", "logicalDump", "options", "source", "connection").(map[string]interface{})
	assert.Equal(t, "db.example.com", conn["host"])
	assert.EqualValues(t, 5432, conn["port"])
	assert.Equal(t, "shop", conn["dbname"])
	assert.Equal(t, "alice", conn["username"])
	assert.Equal(t, "secret", conn["password"])

	assert.Equal(t, "registry.gitlab.com/postgres-ai/se-images/rds:16-0.8.0",
		dig(t, m, "databaseContainer", "dockerImage"))
	assert.Equal(t, "4GB", dig(t, m, "databaseConfigs", "configs", sharedBuffersKey))
	assert.Equal(t, preloadLibs, dig(t, m, "databaseConfigs", "configs", sharedPreloadLibrariesKey))
	assert.Equal(t, "64MB", dig(t, m, "databaseConfigs", "configs", "work_mem"))
	assert.Equal(t, "1.1", dig(t, m, "databaseConfigs", "configs", "random_page_cost"))

	databases := dig(t, m, "retrieval", "spec", "logicalDump", "options", "databases").(map[string]interface{})
	assert.Contains(t, databases, "shop")
}

func TestBuildProjection_ConnectionString(t *testing.T) {
	const url = "postgres://alice@db.example.com:5432/shop?sslmode=require"

	raw, err := buildProjection(sampleProposal(), installOptions{sourceURL: url, password: "secret"})
	require.NoError(t, err)

	m := projectionMap(t, raw)
	connStr := dig(t, m, "retrieval", "spec", "logicalDump", "options", "source", "connectionString")
	assert.Equal(t, url, connStr)
	assert.NotContains(t, connStr, "secret", "the password must never be embedded in the persisted connectionString")

	conn := dig(t, m, "retrieval", "spec", "logicalDump", "options", "source", "connection").(map[string]interface{})
	assert.Equal(t, "secret", conn["password"])
	// discrete fields are written too (consistent with the connection string, not contradictory)
	assert.Equal(t, "db.example.com", conn["host"])
	assert.EqualValues(t, 5432, conn["port"])
	assert.Equal(t, "shop", conn["dbname"])
	assert.Equal(t, "alice", conn["username"])
}

func TestBuildProjection_Overrides(t *testing.T) {
	raw, err := buildProjection(sampleProposal(), installOptions{
		sourceURL:     "postgres://alice@db.example.com:5432/shop",
		dockerImage:   "custom/postgres:99",
		sharedBuffers: "8GB",
		dbnames:       []string{"a", "b"},
	})
	require.NoError(t, err)

	m := projectionMap(t, raw)
	assert.Equal(t, "custom/postgres:99", dig(t, m, "databaseContainer", "dockerImage"))
	assert.Equal(t, "8GB", dig(t, m, "databaseConfigs", "configs", sharedBuffersKey))
	assert.Equal(t, preloadLibs, dig(t, m, "databaseConfigs", "configs", sharedPreloadLibrariesKey))

	databases := dig(t, m, "retrieval", "spec", "logicalDump", "options", "databases").(map[string]interface{})
	assert.Contains(t, databases, "a")
	assert.Contains(t, databases, "b")
	assert.NotContains(t, databases, "shop")
}

func TestDatabaseConfigs(t *testing.T) {
	tests := []struct {
		name     string
		proposal *models.ProposedConfig
		opts     installOptions
		want     map[string]string
	}{
		{
			name: "full proposal", proposal: sampleProposal(),
			want: map[string]string{
				sharedBuffersKey: "4GB", sharedPreloadLibrariesKey: preloadLibs, "work_mem": "64MB", "random_page_cost": "1.1",
			},
		},
		{
			name: "explicit shared_buffers wins", proposal: sampleProposal(), opts: installOptions{sharedBuffers: "8GB"},
			want: map[string]string{
				sharedBuffersKey: "8GB", sharedPreloadLibrariesKey: preloadLibs, "work_mem": "64MB", "random_page_cost": "1.1",
			},
		},
		{
			name: "explicit preload libraries win", proposal: sampleProposal(), opts: installOptions{sharedPreloadLibraries: "auto_explain"},
			want: map[string]string{
				sharedBuffersKey: "4GB", sharedPreloadLibrariesKey: "auto_explain", "work_mem": "64MB", "random_page_cost": "1.1",
			},
		},
		{name: "empty proposal", proposal: &models.ProposedConfig{}, want: map[string]string{}},
		{
			name: "flags only", proposal: &models.ProposedConfig{},
			opts: installOptions{sharedBuffers: "1GB", sharedPreloadLibraries: preloadLibs},
			want: map[string]string{sharedBuffersKey: "1GB", sharedPreloadLibrariesKey: preloadLibs},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, databaseConfigs(tt.proposal, tt.opts))
		})
	}
}

func TestBuildProjection_NoDatabaseConfigs(t *testing.T) {
	opts := installOptions{sourceURL: "postgres://alice@db.example.com:5432/shop"}

	raw, err := buildProjection(&models.ProposedConfig{}, opts)
	require.NoError(t, err)

	_, has := projectionMap(t, raw)["databaseConfigs"]
	assert.False(t, has, "an empty configs block must not be written")
}

func TestComposeImage(t *testing.T) {
	tests := []struct {
		name     string
		resolved string
		image    string
		tag      string
		want     string
	}{
		{name: "resolved used by default", resolved: "repo:16-0.7.0", want: "repo:16-0.7.0"},
		{name: "tag override replaces resolved tag", resolved: "repo:16-0.7.0", tag: "16-glibc236", want: "repo:16-glibc236"},
		{name: "image override wins", resolved: "repo:16-0.7.0", image: "custom/img:1", want: "custom/img:1"},
		{name: "image without tag plus tag override", resolved: "repo:16-0.7.0", image: "custom/img", tag: "1", want: "custom/img:1"},
		{name: "tag override wins over image's own tag", resolved: "repo:16-0.7.0", image: "custom/img:base", tag: "1", want: "custom/img:1"},
		{name: "registry port preserved on tag override", resolved: "reg:5000/repo:16", tag: "17", want: "reg:5000/repo:17"},
		{name: "digest base ignores tag override", resolved: "", image: "repo@sha256:abc", tag: "17", want: "repo@sha256:abc"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, composeImage(tc.resolved, tc.image, tc.tag))
		})
	}
}

func TestSourceCarriesExtraParams(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"postgres://alice@db:5432/shop", false},
		{"postgres://alice@db:5432/shop?sslmode=require", true},
		{"host=db port=5432 dbname=shop user=alice", false},
		{"host=db user=alice sslmode=require", true},
		{"host=db user=alice connect_timeout=5", true},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			assert.Equal(t, tc.want, sourceCarriesExtraParams(tc.url))
		})
	}
}

func TestReadConfirmation(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"yes\n", true},
		{"Y\n", true},
		{"YES\n", true},
		{"n\n", false},
		{"\n", false},
		{"maybe\n", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(strings.TrimSpace(tc.input), func(t *testing.T) {
			got, err := readConfirmation(strings.NewReader(tc.input), &strings.Builder{})
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestOptionsFromContext(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("source-url", "", "")
	fs.String("password", "", "")
	fs.String("shared-buffers", "", "")
	fs.String("shared-preload-libraries", "", "")
	fs.Bool("start", false, "")
	fs.Bool("no-start", false, "")

	require.NoError(t, fs.Set("source-url", "postgres://x@y/z"))
	require.NoError(t, fs.Set("password", "secret"))
	require.NoError(t, fs.Set("shared-buffers", "8GB"))
	require.NoError(t, fs.Set("shared-preload-libraries", preloadLibs))
	require.NoError(t, fs.Set("start", "true"))

	opts := optionsFromContext(cli.NewContext(&cli.App{}, fs, nil))
	assert.Equal(t, "postgres://x@y/z", opts.sourceURL)
	assert.Equal(t, "secret", opts.password)
	assert.Equal(t, "8GB", opts.sharedBuffers)
	assert.Equal(t, preloadLibs, opts.sharedPreloadLibraries)
	assert.True(t, opts.start)
	assert.False(t, opts.noStart)
}

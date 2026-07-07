/*
2026 © Postgres.ai
*/

package probe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImageTag(t *testing.T) {
	tests := []struct {
		tag    string
		want   imageTag
		wantOK bool
	}{
		{tag: "16", want: imageTag{raw: "16", major: 16}, wantOK: true},
		{tag: "16-0.7.0", want: imageTag{raw: "16-0.7.0", major: 16, extVersion: "0.7.0"}, wantOK: true},
		{tag: "16-0.7.0-glibc236", want: imageTag{raw: "16-0.7.0-glibc236", major: 16, extVersion: "0.7.0", glibc: "236"}, wantOK: true},
		{tag: "9.6", wantOK: false},
		{tag: "15beta4", wantOK: false},
		{tag: "16rc1", wantOK: false},
		{tag: "13-nik-test", wantOK: false},
		{tag: "15-fix-ci", wantOK: false},
		{tag: "latest", wantOK: false},
		{tag: "", wantOK: false},
		{tag: "16-0.7", wantOK: false},
		{tag: "16-glibc236", want: imageTag{raw: "16-glibc236", major: 16, glibc: "236"}, wantOK: true},
	}

	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			got, ok := parseImageTag(tc.tag)
			require.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestSelectImageTag(t *testing.T) {
	tags := []string{"16", "16-0.6.4", "16-0.7.0", "16-0.7.0-glibc236", "16-0.7.0-glibc238", "15-0.8.0", "13-nik-test", "9.6"}

	t.Run("glibc hit picks the matching suffix", func(t *testing.T) {
		got, ok := selectImageTag(tags, 16, "236")
		require.True(t, ok)
		assert.Equal(t, "16-0.7.0-glibc236", got)
	})

	t.Run("glibc miss returns false for the glibc-constrained query", func(t *testing.T) {
		_, ok := selectImageTag(tags, 16, "999")
		assert.False(t, ok)
	})

	t.Run("non-glibc picks newest ext version", func(t *testing.T) {
		got, ok := selectImageTag(tags, 16, "")
		require.True(t, ok)
		assert.Equal(t, "16-0.7.0", got, "newest ext version, excluding glibc-suffixed tags")
	})

	t.Run("bare major chosen when it is the only non-glibc tag", func(t *testing.T) {
		got, ok := selectImageTag([]string{"17", "17-0.7.0-glibc236"}, 17, "")
		require.True(t, ok)
		assert.Equal(t, "17", got)
	})

	t.Run("ext-versioned outranks bare major", func(t *testing.T) {
		got, ok := selectImageTag([]string{"17", "17-0.7.0"}, 17, "")
		require.True(t, ok)
		assert.Equal(t, "17-0.7.0", got)
	})

	t.Run("missing major returns false", func(t *testing.T) {
		_, ok := selectImageTag(tags, 99, "")
		assert.False(t, ok)
	})

	t.Run("branch and pre-release tags are ignored", func(t *testing.T) {
		got, ok := selectImageTag([]string{"13-nik-test", "13-fix-ci", "13-0.8.0"}, 13, "")
		require.True(t, ok)
		assert.Equal(t, "13-0.8.0", got)
	})
}

func TestCompareExtVersion(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"0.7.0", "0.6.4", 1},
		{"0.6.4", "0.7.0", -1},
		{"0.8.0", "0.8.0", 0},
		{"", "0.1.0", -1},
		{"0.1.0", "", 1},
		{"", "", 0},
		{"0.10.0", "0.9.0", 1},
	}

	for _, tc := range tests {
		t.Run(tc.a+"_vs_"+tc.b, func(t *testing.T) {
			assert.Equal(t, tc.want, compareExtVersion(tc.a, tc.b))
		})
	}
}

func TestNormalizeGlibcSuffix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"2.36", "236"},
		{"2.31", "231"},
		{"", ""},
		{"153.120", "153120"},
		{"not-a-version", ""},
		{"2.36.1", "2361"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeGlibcSuffix(tc.in))
		})
	}
}

func TestProviderRepo(t *testing.T) {
	tests := []struct {
		provider     string
		wantImageRef string
		wantAPIName  string
		wantBackend  registryBackend
	}{
		{"generic", "postgresai/extended-postgres", "postgresai/extended-postgres", backendDockerHub},
		{"", "postgresai/extended-postgres", "postgresai/extended-postgres", backendDockerHub},
		{"azure", "postgresai/extended-postgres", "postgresai/extended-postgres", backendDockerHub},
		{"rds", "registry.gitlab.com/postgres-ai/se-images/rds", "rds", backendGitLab},
		{"aurora", "registry.gitlab.com/postgres-ai/se-images/aurora", "aurora", backendGitLab},
		{"supabase", "registry.gitlab.com/postgres-ai/se-images/supabase", "supabase", backendGitLab},
		{"heroku", "registry.gitlab.com/postgres-ai/se-images/heroku", "heroku", backendGitLab},
		{"cloudsql", "registry.gitlab.com/postgres-ai/se-images/google-cloud-sql", "google-cloud-sql", backendGitLab},
		{"timescale", "registry.gitlab.com/postgres-ai/se-images/timescale-cloud", "timescale-cloud", backendGitLab},
	}

	for _, tc := range tests {
		t.Run(tc.provider, func(t *testing.T) {
			got := providerRepo(tc.provider)
			assert.Equal(t, tc.wantImageRef, got.imageRef)
			assert.Equal(t, tc.wantAPIName, got.apiName)
			assert.Equal(t, tc.wantBackend, got.backend)
		})
	}
}

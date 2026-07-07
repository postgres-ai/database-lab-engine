/*
2026 © Postgres.ai
*/

package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRegistry builds a Registry whose backends point at the given test
// server and whose offline snapshot is the committed images_fallback.json.
func newTestRegistry(srv *httptest.Server) *Registry {
	return &Registry{
		httpClient:      srv.Client(),
		dockerHubURL:    srv.URL,
		gitlabURL:       srv.URL,
		gitlabProjectID: "1",
		ttl:             time.Hour,
		fetchTimeout:    5 * time.Second,
		cache:           map[string]cacheEntry{},
		fallback:        loadFallback(),
	}
}

// dockerHubHandler serves paginated Docker Hub tag pages for the generic repo,
// counting requests so cache behaviour can be asserted.
func dockerHubHandler(t *testing.T, pages [][]string, hits *int, srv **httptest.Server) http.Handler {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/repositories/"+dockerHubGenericRepo+"/tags", func(w http.ResponseWriter, r *http.Request) {
		*hits++

		idx := 0
		if p := r.URL.Query().Get("page"); p != "" {
			n, _ := strconv.Atoi(p)
			idx = n - 1
		}

		results := make([]map[string]string, 0, len(pages[idx]))
		for _, name := range pages[idx] {
			results = append(results, map[string]string{"name": name})
		}

		next := ""
		if idx+1 < len(pages) {
			next = fmt.Sprintf("%s/v2/repositories/%s/tags?page=%d", (*srv).URL, dockerHubGenericRepo, idx+2)
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{"next": next, "results": results})
	})

	return mux
}

func TestRegistry_DockerHubResolve(t *testing.T) {
	var srv *httptest.Server

	hits := 0
	pages := [][]string{
		{"16-0.7.0", "16-0.7.0-glibc236", "latest"},
		{"16", "15-0.8.0", "13-nik-test"},
	}
	srv = httptest.NewServer(dockerHubHandler(t, pages, &hits, &srv))
	defer srv.Close()

	reg := newTestRegistry(srv)
	ctx := context.Background()

	t.Run("glibc hit selects matching tag across pages", func(t *testing.T) {
		ref, tag := reg.ResolveImage(ctx, "generic", 16, "2.36")
		assert.Equal(t, "postgresai/extended-postgres:16-0.7.0-glibc236", ref)
		assert.Equal(t, "16-0.7.0-glibc236", tag)
	})

	t.Run("second call is served from the fresh cache", func(t *testing.T) {
		before := hits
		_, _ = reg.ResolveImage(ctx, "generic", 16, "2.36")
		assert.Equal(t, before, hits, "fresh cache must not re-hit the registry")
	})

	t.Run("glibc miss falls back to newest non-glibc tag", func(t *testing.T) {
		ref, tag := reg.ResolveImage(ctx, "generic", 16, "2.99")
		assert.Equal(t, "postgresai/extended-postgres:16-0.7.0", ref)
		assert.Equal(t, "16-0.7.0", tag)
	})

	t.Run("missing major returns the provider default and empty tag", func(t *testing.T) {
		ref, tag := reg.ResolveImage(ctx, "generic", 99, "2.36")
		assert.Equal(t, "postgresai/extended-postgres:99", ref)
		assert.Empty(t, tag)
	})

	t.Run("PG<15 ignores collation and picks newest non-glibc", func(t *testing.T) {
		ref, tag := reg.ResolveImage(ctx, "generic", 16, "")
		assert.Equal(t, "postgresai/extended-postgres:16-0.7.0", ref)
		assert.Equal(t, "16-0.7.0", tag)
	})
}

func TestRegistry_GitLabResolve(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/1/registry/repositories", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"id": 10, "name": "rds"}})
	})
	mux.HandleFunc("/api/v4/projects/1/registry/repositories/10/tags", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page != "" && page != "1" {
			_ = json.NewEncoder(w).Encode([]map[string]string{})
			return
		}

		_ = json.NewEncoder(w).Encode([]map[string]string{{"name": "15-0.8.0"}, {"name": "14-0.8.0"}, {"name": "15-fix-ci"}})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	reg := newTestRegistry(srv)

	ref, tag := reg.ResolveImage(context.Background(), "rds", 15, "2.36")
	assert.Equal(t, "registry.gitlab.com/postgres-ai/se-images/rds:15-0.8.0", ref)
	assert.Equal(t, "15-0.8.0", tag)
}

func TestRegistry_GitLabTagPagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/1/registry/repositories", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"id": 10, "name": "rds"}})
	})
	mux.HandleFunc("/api/v4/projects/1/registry/repositories/10/tags", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "", "1":
			// a full page keeps the loop going to page 2
			full := make([]map[string]string, 0, registryPageSize)
			for i := 0; i < registryPageSize; i++ {
				full = append(full, map[string]string{"name": "15-0.6.0"})
			}

			_ = json.NewEncoder(w).Encode(full)

		case "2":
			_ = json.NewEncoder(w).Encode([]map[string]string{{"name": "15-0.9.0"}})

		default:
			_ = json.NewEncoder(w).Encode([]map[string]string{})
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	reg := newTestRegistry(srv)

	ref, tag := reg.ResolveImage(context.Background(), "rds", 15, "")
	assert.Equal(t, "registry.gitlab.com/postgres-ai/se-images/rds:15-0.9.0", ref,
		"the newest tag lives on page 2, so pagination must traverse it")
	assert.Equal(t, "15-0.9.0", tag)
}

func TestRegistry_DockerHubRebasesNextURL(t *testing.T) {
	var srv *httptest.Server

	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/repositories/"+dockerHubGenericRepo+"/tags", func(w http.ResponseWriter, r *http.Request) {
		hits++

		if r.URL.Query().Get("page") == "2" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"next": "", "results": []map[string]string{{"name": "16-0.9.0"}}})
			return
		}

		// next points at the real hub.docker.com host; rebase must redirect it to the test server.
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"next":    "https://hub.docker.com/v2/repositories/" + dockerHubGenericRepo + "/tags?page=2",
			"results": []map[string]string{{"name": "16-0.7.0"}},
		})
	})

	srv = httptest.NewServer(mux)
	defer srv.Close()

	reg := newTestRegistry(srv)

	ref, tag := reg.ResolveImage(context.Background(), "generic", 16, "")
	assert.Equal(t, 2, hits, "the rebased next URL must be followed on the test server, not hub.docker.com")
	assert.Equal(t, "postgresai/extended-postgres:16-0.9.0", ref)
	assert.Equal(t, "16-0.9.0", tag)
}

func TestRegistry_FallbackOnError(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer srv.Close()

			reg := newTestRegistry(srv)

			// the embedded snapshot lists generic 16-0.7.0, so resolution still
			// succeeds without any reachable registry.
			ref, tag := reg.ResolveImage(context.Background(), "generic", 16, "")
			assert.Equal(t, "postgresai/extended-postgres:16-0.7.0", ref)
			assert.Equal(t, "16-0.7.0", tag)
		})
	}
}

func TestRegistry_FallbackThenDefaultWhenSnapshotLacksMajor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	reg := newTestRegistry(srv)

	// PG21 is absent from both the dead registry and the snapshot → default ref.
	ref, tag := reg.ResolveImage(context.Background(), "generic", 21, "")
	assert.Equal(t, "postgresai/extended-postgres:21", ref)
	assert.Empty(t, tag)
}

func TestRegistry_AzureFallsBackToGenericImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	reg := newTestRegistry(srv)

	ref, _ := reg.ResolveImage(context.Background(), "azure", 16, "")
	require.Contains(t, ref, dockerHubGenericRepo, "azure has no SE repo and must use the generic image")
}

func TestLoadFallback_ParsesEmbeddedSnapshot(t *testing.T) {
	fb := loadFallback()

	require.Contains(t, fb, "postgresai/extended-postgres")
	require.Contains(t, fb, "registry.gitlab.com/postgres-ai/se-images/google-cloud-sql")
	assert.Contains(t, fb["postgresai/extended-postgres"], "16-0.7.0")
}

func TestProviderRepoMatchesFallbackKeys(t *testing.T) {
	// the offline fallback only works when providerRepo(...).imageRef equals a
	// `repo` key in images_fallback.json; this joins the two so a rename of
	// either side fails loudly instead of silently degrading to the default.
	fb := loadFallback()

	for _, key := range []string{"generic", "rds", "aurora", "supabase", "cloudsql", "heroku", "timescale"} {
		repo := providerRepo(key)
		assert.Containsf(t, fb, repo.imageRef,
			"providerRepo(%q).imageRef=%q must be a key in the embedded snapshot", key, repo.imageRef)
	}

	// azure has no SE repo and resolves to the generic image, which is in the snapshot.
	assert.Contains(t, fb, providerRepo("azure").imageRef)
}

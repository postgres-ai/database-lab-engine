/*
2026 © Postgres.ai
*/

package probe

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

//go:embed images_fallback.json
var fallbackJSON []byte

const (
	defaultDockerHubURL    = "https://hub.docker.com"
	defaultGitLabURL       = "https://gitlab.com"
	defaultGitLabProjectID = "41786321"

	defaultRegistryTTL          = time.Hour
	defaultRegistryFetchTimeout = 5 * time.Second
	defaultRegistryHTTPTimeout  = 10 * time.Second

	registryUserAgent = "postgres-ai-dblab-engine (registry image resolver)"

	maxRegistryPages = 50
	registryPageSize = 100
)

// Registry lists image tags for a repo across the Docker Hub (generic image) and
// GitLab (managed-provider SE images) backends, resolving a glibc-aware image
// reference. Results are cached per repo with a TTL; lookups degrade through
// fresh cache → live fetch → last good cache → embedded offline snapshot → the
// provider default, so a probe never hangs on or hard-fails from a slow or
// rate-limited registry, and works fully offline.
type Registry struct {
	httpClient      *http.Client
	dockerHubURL    string
	gitlabURL       string
	gitlabProjectID string

	ttl          time.Duration
	fetchTimeout time.Duration

	mu        sync.RWMutex
	cache     map[string]cacheEntry
	refreshMu sync.Mutex

	fallback map[string][]string
}

type cacheEntry struct {
	tags      []string
	fetchedAt time.Time
}

// NewRegistry builds a Registry pointed at the public Docker Hub and GitLab
// endpoints with the embedded offline snapshot loaded.
func NewRegistry() *Registry {
	return &Registry{
		httpClient:      &http.Client{Timeout: defaultRegistryHTTPTimeout},
		dockerHubURL:    defaultDockerHubURL,
		gitlabURL:       defaultGitLabURL,
		gitlabProjectID: defaultGitLabProjectID,
		ttl:             defaultRegistryTTL,
		fetchTimeout:    defaultRegistryFetchTimeout,
		cache:           map[string]cacheEntry{},
		fallback:        loadFallback(),
	}
}

// ResolveImage resolves a full `<repo>:<tag>` image reference and the selected
// tag for a probe provider key. For PG15+ with a libc collation version it
// prefers the newest tag whose glibc suffix matches; on a glibc miss (or PG<15,
// ICU, empty collation) it picks the newest non-glibc tag; when the major is
// absent from the registry it returns the provider default `<repo>:<major>` and
// an empty tag (so the UI catalog still resolves it). It never returns an empty
// reference.
func (r *Registry) ResolveImage(
	ctx context.Context, providerKey string, major int, collationVersion string,
) (resolvedRef, dockerTag string) {
	repo := providerRepo(providerKey)

	glibcSuffix := ""
	if major >= collationMinMajor {
		glibcSuffix = normalizeGlibcSuffix(collationVersion)
	}

	tags := r.listTags(ctx, repo)

	if tag, ok := selectImageTag(tags, major, glibcSuffix); ok {
		return repo.imageRef + ":" + tag, tag
	}

	if glibcSuffix != "" {
		if tag, ok := selectImageTag(tags, major, ""); ok {
			return repo.imageRef + ":" + tag, tag
		}
	}

	return repo.imageRef + ":" + strconv.Itoa(major), ""
}

// listTags returns the tags for repo, walking the resolution chain. It never
// returns an error: a nil result means even the offline snapshot had no entry,
// leaving the caller to fall back to the provider default.
func (r *Registry) listTags(ctx context.Context, repo repoRef) []string {
	if tags, ok := r.freshCachedTags(repo.imageRef); ok {
		return tags
	}

	if tags := r.refreshTags(ctx, repo); tags != nil {
		return tags
	}

	if tags, ok := r.cachedTags(repo.imageRef); ok {
		return tags
	}

	return r.fallback[repo.imageRef]
}

// refreshTags performs the bounded live fetch under a refresh lock so concurrent
// callers coalesce onto one request. It returns the fetched tags (also cached)
// or nil when the fetch fails or yields nothing.
func (r *Registry) refreshTags(ctx context.Context, repo repoRef) []string {
	r.refreshMu.Lock()
	defer r.refreshMu.Unlock()

	if tags, ok := r.freshCachedTags(repo.imageRef); ok {
		return tags
	}

	fetchCtx, cancel := context.WithTimeout(ctx, r.fetchTimeout)
	defer cancel()

	tags, err := r.fetchTags(fetchCtx, repo)
	if err != nil || len(tags) == 0 {
		return nil
	}

	r.storeCache(repo.imageRef, tags)

	return tags
}

func (r *Registry) freshCachedTags(imageRef string) ([]string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.cache[imageRef]
	if !ok || time.Since(entry.fetchedAt) >= r.ttl {
		return nil, false
	}

	return entry.tags, true
}

func (r *Registry) cachedTags(imageRef string) ([]string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.cache[imageRef]
	if !ok {
		return nil, false
	}

	return entry.tags, true
}

func (r *Registry) storeCache(imageRef string, tags []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache[imageRef] = cacheEntry{tags: tags, fetchedAt: time.Now()}
}

func (r *Registry) fetchTags(ctx context.Context, repo repoRef) ([]string, error) {
	switch repo.backend {
	case backendDockerHub:
		return r.fetchDockerHubTags(ctx, repo.apiName)

	case backendGitLab:
		return r.fetchGitLabTags(ctx, repo.apiName)
	}

	return nil, fmt.Errorf("unknown registry backend for %q", repo.imageRef)
}

type dockerHubTagsResponse struct {
	Next    string `json:"next"`
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
}

func (r *Registry) fetchDockerHubTags(ctx context.Context, repo string) ([]string, error) {
	url := fmt.Sprintf("%s/v2/repositories/%s/tags?page_size=%d", r.dockerHubURL, repo, registryPageSize)

	var tags []string

	for i := 0; i < maxRegistryPages && url != ""; i++ {
		var resp dockerHubTagsResponse

		if err := r.getJSON(ctx, url, &resp); err != nil {
			return nil, err
		}

		for _, res := range resp.Results {
			tags = append(tags, res.Name)
		}

		// Docker Hub returns `next` as an absolute hub.docker.com URL; rebase it
		// onto the configured host so an overridden base (mirror/test) keeps
		// paginating against that base instead of escaping to the internet.
		url = rebaseURL(resp.Next, r.dockerHubURL)
	}

	return tags, nil
}

// rebaseURL keeps the path and query of next but swaps in base's scheme and
// host. An empty or unparseable next yields "" so pagination stops.
func rebaseURL(next, base string) string {
	if next == "" {
		return ""
	}

	nextURL, err := url.Parse(next)
	if err != nil {
		return ""
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return next
	}

	nextURL.Scheme = baseURL.Scheme
	nextURL.Host = baseURL.Host

	return nextURL.String()
}

type gitlabRepo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type gitlabTag struct {
	Name string `json:"name"`
}

func (r *Registry) fetchGitLabTags(ctx context.Context, repoName string) ([]string, error) {
	repoID, err := r.gitlabRepoID(ctx, repoName)
	if err != nil {
		return nil, err
	}

	var tags []string

	for page := 1; page <= maxRegistryPages; page++ {
		url := fmt.Sprintf("%s/api/v4/projects/%s/registry/repositories/%d/tags?per_page=%d&page=%d",
			r.gitlabURL, r.gitlabProjectID, repoID, registryPageSize, page)

		var pageTags []gitlabTag

		if err := r.getJSON(ctx, url, &pageTags); err != nil {
			return nil, err
		}

		for _, t := range pageTags {
			tags = append(tags, t.Name)
		}

		if len(pageTags) < registryPageSize {
			break
		}
	}

	return tags, nil
}

func (r *Registry) gitlabRepoID(ctx context.Context, repoName string) (int, error) {
	for page := 1; page <= maxRegistryPages; page++ {
		reqURL := fmt.Sprintf("%s/api/v4/projects/%s/registry/repositories?per_page=%d&page=%d",
			r.gitlabURL, r.gitlabProjectID, registryPageSize, page)

		var repos []gitlabRepo

		if err := r.getJSON(ctx, reqURL, &repos); err != nil {
			return 0, err
		}

		for _, repo := range repos {
			if repo.Name == repoName {
				return repo.ID, nil
			}
		}

		if len(repos) < registryPageSize {
			break
		}
	}

	return 0, fmt.Errorf("gitlab repo %q not found in project %s", repoName, r.gitlabProjectID)
}

// getJSON performs a GET and decodes a 200 response into out. Any non-200 status
// (including 429/5xx) is an error so the caller serves cache/fallback instead.
func (r *Registry) getJSON(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", registryUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry returned status %d for %s", resp.StatusCode, url)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode registry response: %w", err)
	}

	return nil
}

type fallbackSnapshot struct {
	Providers []struct {
		Provider string   `json:"provider"`
		Registry string   `json:"registry"`
		Repo     string   `json:"repo"`
		Tags     []string `json:"tags"`
	} `json:"providers"`
}

// loadFallback parses the embedded offline snapshot into a repo→tags map. The
// snapshot is committed and tested, so a parse failure yields an empty map
// rather than a panic (live/cache layers still work).
func loadFallback() map[string][]string {
	out := map[string][]string{}

	var snap fallbackSnapshot
	if err := json.Unmarshal(fallbackJSON, &snap); err != nil {
		return out
	}

	for _, p := range snap.Providers {
		out[p.Repo] = p.Tags
	}

	return out
}

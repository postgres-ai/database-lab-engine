/*
2026 © Postgres.ai
*/

package probe

import (
	"regexp"
	"strconv"
	"strings"
)

// registryBackend identifies which registry hosts a repo's tags.
type registryBackend int

const (
	backendDockerHub registryBackend = iota
	backendGitLab
)

const (
	// dockerHubGenericRepo is the generic extended-postgres image on Docker Hub.
	dockerHubGenericRepo = "postgresai/extended-postgres"

	// gitlabSEPrefix is the image-reference prefix for the managed-provider SE
	// images on the GitLab container registry.
	gitlabSEPrefix = "registry.gitlab.com/postgres-ai/se-images/"
)

// providerToSERepo maps the probe provider keys whose SE repo name differs from
// the key itself (mirrors the UI's providerKeyToImageType). All other managed
// keys use their key verbatim as the repo name.
var providerToSERepo = map[string]string{
	"cloudsql":  "google-cloud-sql",
	"timescale": "timescale-cloud",
}

// providersWithoutSERepo have no managed SE repo and resolve to the generic
// image instead.
var providersWithoutSERepo = map[string]bool{
	"azure": true,
}

// repoRef locates a repo for both the resolved image reference and the registry
// tags API. imageRef is the prefix of the pullable image (`<imageRef>:<tag>`);
// apiName is the identifier the backend's tags API keys on (the full Docker Hub
// repo, or the bare GitLab repo name).
type repoRef struct {
	imageRef string
	apiName  string
	backend  registryBackend
}

// providerRepo maps a probe provider key to its registry repo. The generic key,
// the empty key, and any provider without an SE repo (e.g. azure) resolve to the
// generic Docker Hub image; every other key resolves to its GitLab SE repo.
func providerRepo(providerKey string) repoRef {
	if providerKey == "" || providerKey == string(ProviderGeneric) || providersWithoutSERepo[providerKey] {
		return repoRef{imageRef: dockerHubGenericRepo, apiName: dockerHubGenericRepo, backend: backendDockerHub}
	}

	repoName := providerKey
	if mapped, ok := providerToSERepo[providerKey]; ok {
		repoName = mapped
	}

	return repoRef{imageRef: gitlabSEPrefix + repoName, apiName: repoName, backend: backendGitLab}
}

// imageTag is a tag parsed under the unified scheme
// `<pgMajor>[-<extVersion>][-glibc<NNN>]`.
type imageTag struct {
	raw        string
	major      int
	extVersion string // "" when the tag is a bare major
	glibc      string // the NNN digits, "" when the tag carries no glibc suffix
}

// tagPattern matches the strict release-tag form. The leading-integer-only major
// drops EOL dotted majors (9.6), pre-releases (15beta4, 16rc1), and SE
// branch-named CI tags (13-nik-…, 15-fix-ci) automatically.
var tagPattern = regexp.MustCompile(`^(\d+)(?:-(\d+\.\d+\.\d+))?(?:-glibc(\d+))?$`)

// parseImageTag parses a single registry tag. It returns false for tags that do
// not match the strict scheme, so non-release tags are ignored by the selector.
func parseImageTag(tag string) (imageTag, bool) {
	m := tagPattern.FindStringSubmatch(tag)
	if m == nil {
		return imageTag{}, false
	}

	major, err := strconv.Atoi(m[1])
	if err != nil {
		return imageTag{}, false
	}

	return imageTag{raw: tag, major: major, extVersion: m[2], glibc: m[3]}, true
}

// selectImageTag picks the best tag for major from tags. When glibcSuffix is set
// only tags carrying the matching glibc suffix are considered; otherwise only
// non-glibc tags are considered. Among candidates the newest extVersion wins; a
// bare-major tag ranks below any ext-versioned tag. Returns ("", false) when no
// tag matches the major under the given constraint.
func selectImageTag(tags []string, major int, glibcSuffix string) (string, bool) {
	var (
		best  imageTag
		found bool
	)

	for _, raw := range tags {
		t, ok := parseImageTag(raw)
		if !ok || t.major != major {
			continue
		}

		if glibcSuffix != "" && t.glibc != glibcSuffix {
			continue
		}

		if glibcSuffix == "" && t.glibc != "" {
			continue
		}

		if !found || compareExtVersion(t.extVersion, best.extVersion) > 0 {
			best = t
			found = true
		}
	}

	if !found {
		return "", false
	}

	return best.raw, true
}

// compareExtVersion compares two "x.y.z" ext-version strings, returning -1, 0, or
// 1. An empty string (bare-major tag) sorts below any concrete version.
func compareExtVersion(a, b string) int {
	if a == b {
		return 0
	}

	if a == "" {
		return -1
	}

	if b == "" {
		return 1
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		ai, _ := strconv.Atoi(aParts[i])
		bi, _ := strconv.Atoi(bParts[i])

		if ai < bi {
			return -1
		}

		if ai > bi {
			return 1
		}
	}

	return signOf(len(aParts) - len(bParts))
}

// signOf clamps n to -1, 0, or 1 so compareExtVersion honours its documented
// contract even for unequal component counts.
func signOf(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

// normalizeGlibcSuffix turns a probed libc collation (glibc) version into the
// glibc tag suffix digits (e.g. "2.36" → "236"). The probe supplies a libc
// version or an empty string (ICU/builtin and C/POSIX already resolve to empty
// upstream), so an empty or non-numeric input returns "", disabling glibc
// matching for that source.
func normalizeGlibcSuffix(collationVersion string) string {
	if collationVersion == "" {
		return ""
	}

	suffix := strings.ReplaceAll(collationVersion, ".", "")

	for _, r := range suffix {
		if r < '0' || r > '9' {
			return ""
		}
	}

	return suffix
}

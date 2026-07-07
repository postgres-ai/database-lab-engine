/*
2026 © Postgres.ai
*/

package probe

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	// SHOW returns text; matching the integer cast in tools/db/pg.go keeps the
	// scan path simple and lets pgx decode directly into int.
	versionQuery       = `select setting::int/10000 from pg_settings where name = 'server_version_num'`
	preloadLibsQuery   = `show shared_preload_libraries`
	availableExtsQuery = `select name from pg_available_extensions`

	// collationVersionQuery reads the connected database's recorded collation
	// version, but only for libc-provider databases (datlocprovider = 'c') —
	// those are the ones whose collation version is a glibc version that can
	// drive image selection. ICU/builtin providers report a non-glibc version
	// that must not be treated as a glibc suffix, so they map to an empty
	// string. pg_database.datcollversion and datlocprovider exist only on PG15+,
	// so this query must be guarded by the major-version check; coalesce maps a
	// NULL (C/POSIX locale) to an empty string.
	collationVersionQuery = `select case when datlocprovider = 'c' then coalesce(datcollversion, '') else '' end ` +
		`from pg_database where datname = current_database()`

	// collationMinMajor is the first PostgreSQL major version exposing
	// pg_database.datcollversion and datlocprovider.
	collationMinMajor = 15
)

// Propose connects to the source described by connStr (URI or DSN; no password
// permitted in the string), inspects it, and returns a structured proposal
// suitable for a Simple-mode preview. The connection is opened with the
// supplied password injected separately so it never appears in the parsed
// connection string. The host's /proc/meminfo informs the shared_buffers
// recommendation.
//
// Returns a wrapped error on parse failures, connection failures, or unexpected
// query failures. Missing optional data (e.g. an empty shared_preload_libraries
// on the source) is tolerated and surfaces as an empty field in the result.
//
// reg resolves the docker image from the live registries (with offline
// fallback). When reg is nil image resolution is skipped and ResolvedImage /
// DockerTag stay empty (the UI catalog then resolves the image).
func Propose(ctx context.Context, connStr, password string, reg *Registry) (ProposedConfig, error) {
	conn, err := ParseConnectionString(connStr)
	if err != nil {
		return ProposedConfig{}, err
	}

	cfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		return ProposedConfig{}, fmt.Errorf("rebuild connection config: %w", err)
	}

	cfg.Password = password

	pgConn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return ProposedConfig{}, fmt.Errorf("connect to source: %w", err)
	}

	defer func() { _ = pgConn.Close(ctx) }()

	major, err := queryMajorVersion(ctx, pgConn)
	if err != nil {
		return ProposedConfig{}, err
	}

	collationVersion, err := queryCollationVersion(ctx, pgConn, major)
	if err != nil {
		return ProposedConfig{}, err
	}

	sourceLibs, err := queryPreloadLibs(ctx, pgConn)
	if err != nil {
		return ProposedConfig{}, err
	}

	exts, err := queryAvailableExtensions(ctx, pgConn)
	if err != nil {
		return ProposedConfig{}, err
	}

	tuning, err := CollectTuningParams(ctx, pgConn)
	if err != nil {
		return ProposedConfig{}, err
	}

	hostMem := DetectHostMemoryBytes(os.DirFS("/"))

	resolvedImage, dockerTag := resolveImage(ctx, reg, DetectProvider(conn.Host, exts), major, collationVersion)

	return assembleProposed(proposalInputs{
		conn:                conn,
		pgMajorVersion:      major,
		collationVersion:    collationVersion,
		sourceLibs:          sourceLibs,
		availableExtensions: exts,
		tuning:              tuning,
		hostMemBytes:        hostMem,
		resolvedImage:       resolvedImage,
		dockerTag:           dockerTag,
	}), nil
}

// resolveImage asks the registry for a concrete image reference and tag. It is a
// no-op (empty results) when reg is nil so callers without a registry — and the
// integration tests — keep working.
func resolveImage(
	ctx context.Context, reg *Registry, provider Provider, major int, collationVersion string,
) (resolvedImage, dockerTag string) {
	if reg == nil {
		return "", ""
	}

	return reg.ResolveImage(ctx, string(provider), major, collationVersion)
}

// proposalInputs carries the raw signals collected from the source and the host
// that assembleProposed combines into a ProposedConfig.
type proposalInputs struct {
	conn                Connection
	pgMajorVersion      int
	collationVersion    string
	sourceLibs          []string
	availableExtensions []string
	tuning              map[string]string
	hostMemBytes        uint64
	resolvedImage       string
	dockerTag           string
}

// assembleProposed turns the raw inputs collected from the source and the host
// into a ProposedConfig. Factored out so unit tests can exercise the combinator
// logic without standing up a Postgres instance.
func assembleProposed(in proposalInputs) ProposedConfig {
	provider := DetectProvider(in.conn.Host, in.availableExtensions)

	dbs := []string{}
	if in.conn.DBName != "" {
		dbs = append(dbs, in.conn.DBName)
	}

	return ProposedConfig{
		Source:                 in.conn,
		DetectedProvider:       provider,
		DockerImage:            string(provider),
		DockerTag:              in.dockerTag,
		ResolvedImage:          in.resolvedImage,
		PgMajorVersion:         in.pgMajorVersion,
		CollationVersion:       in.collationVersion,
		Databases:              dbs,
		SharedBuffers:          RecommendSharedBuffers(in.hostMemBytes),
		MemoryProbed:           in.hostMemBytes > 0,
		SharedPreloadLibraries: ResolvePreloadLibs(in.sourceLibs),
		QueryTuning:            in.tuning,
	}
}

func queryMajorVersion(ctx context.Context, conn *pgx.Conn) (int, error) {
	var major int

	if err := conn.QueryRow(ctx, versionQuery).Scan(&major); err != nil {
		return 0, fmt.Errorf("query server_version_num: %w", err)
	}

	return major, nil
}

// queryCollationVersion returns the connected database's recorded libc
// collation (glibc) version. It is a no-op (empty result) below PG15, where the
// queried columns do not exist. An empty value (ICU/builtin provider, C/POSIX
// locale, NULL, or no collation recorded) is tolerated and returned as the
// empty string, not an error.
func queryCollationVersion(ctx context.Context, conn *pgx.Conn, pgMajorVersion int) (string, error) {
	if pgMajorVersion < collationMinMajor {
		return "", nil
	}

	var version string

	if err := conn.QueryRow(ctx, collationVersionQuery).Scan(&version); err != nil {
		return "", fmt.Errorf("query datcollversion: %w", err)
	}

	return version, nil
}

func queryPreloadLibs(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	var libsCSV string

	if err := conn.QueryRow(ctx, preloadLibsQuery).Scan(&libsCSV); err != nil {
		return nil, fmt.Errorf("query shared_preload_libraries: %w", err)
	}

	if libsCSV == "" {
		return nil, nil
	}

	parts := strings.Split(libsCSV, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}

	return out, nil
}

func queryAvailableExtensions(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	rows, err := conn.Query(ctx, availableExtsQuery)
	if err != nil {
		return nil, fmt.Errorf("query pg_available_extensions: %w", err)
	}

	defer rows.Close()

	out := []string{}

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan pg_available_extensions row: %w", err)
		}

		out = append(out, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pg_available_extensions rows: %w", err)
	}

	return out, nil
}

/*
2026 © Postgres.ai
*/

// Package probe inspects a source PostgreSQL instance and the DBLab host to
// propose a retrieval configuration. The package is HTTP- and UI-free; callers
// are the /admin/probe-source HTTP handler and the dblab local-install CLI,
// which calls probe.Propose directly without going over HTTP.
package probe

import "errors"

// ErrPasswordInConnString is returned by ParseConnectionString when the
// supplied URI or DSN embeds a password. Callers must pass the password as a
// separate value so it never reaches logs or telemetry.
var ErrPasswordInConnString = errors.New("password must not be embedded in the connection string")

// ErrMultiHostConnString is returned by ParseConnectionString when the supplied
// URI or DSN lists more than one host. Multi-host failover targets are out of
// scope for the simplified install flow.
var ErrMultiHostConnString = errors.New("multi-host connection strings are not supported")

// Connection mirrors the shape of logical.Connection so callers can map the
// parsed result onto retrieval configs without depending on the logical
// package. Password is intentionally absent — it is supplied as a separate
// argument throughout the probe API.
type Connection struct {
	Host     string
	Port     int
	Username string
	DBName   string
}

// ProposedConfig is the structured proposal returned by Propose for a given
// source. The Simple-mode preview renders it directly; the /admin/probe-source
// HTTP handler serialises it as JSON. The shape intentionally carries no
// free-form Warnings slice — UI copy is generated from structured signals
// (DetectedProvider, MemoryProbed) so the engine response stays a pure data
// contract.
//
// DockerImage carries the provider key (e.g. "rds", "supabase") for backward
// compatibility; the UI resolves the concrete docker-image id from its catalog
// when ResolvedImage is empty. DockerTag is the registry tag selected by the
// engine-side resolver (empty when no registry tag matched, letting the UI
// catalog resolve it). ResolvedImage is the full `<repo>:<tag>` reference the
// engine resolved from the live registries (with offline fallback); when set,
// callers should use it directly rather than re-deriving from the provider key.
type ProposedConfig struct {
	Source           Connection
	DetectedProvider Provider
	DockerImage      string
	DockerTag        string
	ResolvedImage    string
	PgMajorVersion   int
	// CollationVersion is the source database's recorded libc collation (glibc)
	// version (pg_database.datcollversion for datlocprovider='c'), available on
	// PG15+. It drives glibc-aware image selection (e.g. "2.36"). It is empty on
	// PG<15, for ICU/builtin providers, on C/POSIX locales, or when otherwise
	// unavailable. It is a strong signal, not a literal ldd glibc reading.
	CollationVersion       string
	Databases              []string
	SharedBuffers          string
	MemoryProbed           bool
	SharedPreloadLibraries string
	QueryTuning            map[string]string
}

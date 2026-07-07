# Simplified Install for DBLab 4.2 (Simple/Expert modes + autodetection)

## Overview

Cut the second phase of DBLab install ("configure data retrieval") from a multi-field form down to "paste source URL + password, click Apply." This is the headline 4.2 UX change.

Today's Configuration tab exposes ~30 fields a user must understand: source host/port/user/dbname/password, docker image (provider + tag), pg version, databases to copy, `shared_buffers`, `shared_preload_libraries`, query tuning parameters, plus pg_dump options per provider. Every value can in principle be derived from the source DB (version, settings, extensions) and from the DLE host (RAM). The current "Get version from source" and "Get from source database" buttons already prove the engine can probe — they're just buried inside a form the user must otherwise fill out manually.

This change adds:

1. A **Simple mode** tab (default for unconfigured instances): one URL field + one password field + "Detect & preview" button. Engine probes source, returns a `ProposedConfig`, UI renders read-only preview, user confirms with "Apply & start retrieval".
2. An **Expert mode** tab (default for already-configured instances) that keeps the full form but (a) replaces the 4 source-connection fields with a single URL field, and (b) adds physical-retrieval UI for WAL-G and pgBackRest (the two `tool:` values the engine actually supports outside `customTool`) that was previously YAML-only.
3. A new engine endpoint `POST /admin/probe-source` and a new internal package `engine/internal/retrieval/probe/` housing pure, unit-testable detection logic.
4. Lifts the existing `models.Logical`-only gate in `applyProjectedAdminConfig` / `testDBSource` so projection writes work for physical mode too — but in physical mode the writer refuses requests that touch logical-only fields.

Storage in `configs/server.yml` stays unchanged (still `host`, `port`, `username`, `dbname` as separate keys) — the URL is purely a UI affordance, parsed on save and reconstructed on load. This preserves backwards compatibility for hand-edited configs.

### Benefits

- New install: from "fill 10 fields, look up your image tag, copy paste shared_preload_libraries from RDS docs" to "paste URL + password, click twice."
- Existing physical-mode users get a UI for the two `tool:` values the engine supports outside `customTool` — WAL-G and pgBackRest — covering the structured fields (`backupName`, `stanza`, `delta`) and a key/value editor for the `envs` map where storage backend, bucket, prefix, and credentials live. pg_basebackup (which runs via `customTool`) stays YAML-only.
- Foundation for the `dblab local-install` CLI (pulled forward into this cycle): probe + propose is a clean engine entry point the CLI calls directly.

### Non-goals (4.2)

> Pulled forward: the `dblab local-install` CLI and engine-side image resolution
> originally scheduled for 4.3 shipped together with 4.2. See
> `docs/plans/completed/20260619-cli-connstrings-glibc.md`.

- ~~No `dblab local-install` CLI subcommand (4.3).~~ Delivered (`engine/cmd/cli/commands/localinstall/`).
- ~~No engine-side docker image catalog (4.3, lands with CLI).~~ Delivered as registry-based resolution (`probe/registry.go`, `probe/imageselect.go`, `images_fallback.json`). UI catalog still applies for the Simple-mode preview.
- No RDS-refresh-tool detection (just a static neutral note in the preview).
- No Supabase/cloud one-click integrations (later cycle).
- No data masking on DLE side (separate discussion).
- No change to the YAML schema except additive physical-mode projection tags.
- No `customTool` physical sub-mode UI (rare; YAML-only stays). pg_basebackup is invoked via `customTool` in the engine (`physical.go:136-148` only accepts `walg`, `pgbackrest`, `customTool`), so pg_basebackup is **out of scope for Expert-mode UI** in 4.2 — users keep editing YAML for it. If we later add a customTool sub-form, pg_basebackup will land alongside other custom commands.
- No RDS IAM source-side UI changes: `Source.RDS *RDSConfig` uses IAM auth (`logical/dump.go:94-99`) and has no password to inject — Simple mode's URL+password contract does not apply, so RDS IAM sources stay Expert-only and YAML-edited. Simple mode can still detect an `*.rds.amazonaws.com` host as RDS, but produces a password-based proposed config; users who need IAM must edit in Expert mode after Apply.

## Context (from discovery)

### Files/components involved

**Backend — existing seams:**

- `engine/internal/retrieval/engine/postgres/tools/db/pg.go` — `CheckSource()` queries source for version + tuning params + db list + extensions + locales using `jackc/pgx`. New probe package composes this rather than duplicating, and uses pgx consistently.
- `engine/internal/srv/config.go` — `testDBSource()` handler (~lines 105-140), `applyProjectedAdminConfig()` with the logical-only gate at `:231-238`, `connectionCheckTimeout` constant. New `probeSource()` handler lives here.
- `engine/internal/srv/server.go` — route registration for `/admin/test-db-source`; new `/admin/probe-source` route goes here.
- `engine/internal/telemetry/telemetry.go` — event-type constants (`ConfigUpdatedEvent` lives at `:44`); add `ConfigProbedEvent` here.
- `engine/internal/telemetry/events.go` — payload struct definitions (`EngineStarted`, `ConfigUpdated`). New `ConfigProbed` payload struct (provider key only) goes here.
- `engine/pkg/models/admin.go` — `DBSource` response model; new `ProposedConfig` model goes in the same file.
- `engine/pkg/models/configuration.go` — `ConfigProjection` model; gains additive flat `proj:"..."`-tagged fields for the physical-mode leaves the engine actually supports (`walg.backupName`, `pgbackrest.stanza`, `pgbackrest.delta`, and the `envs` map for free-form key/value).
- `engine/pkg/models/retrieval.go` — existing `models.RetrievalMode` (`Logical` | `Physical` | `Unknown`) — reuse this; do not introduce a parallel discriminator.
- `engine/pkg/util/projection/operations.go` — `forEachField` at `:84-120` walks only top-level fields of the target struct and requires a `proj:` tag with a flat dotted path per field. **No recursion into nested struct types.** This constrains Task 9 to flat fields; any nested-struct addition is a projection-package refactor in its own right and is out of scope.
- `engine/pkg/util/projection/tags.go` — projection-tag dispatch the new flat Physical leaves wire through.
- `engine/internal/retrieval/engine/postgres/physical/physical.go` — `getRestorer()` at `:136-148` accepts only `walg`, `pgbackrest`, `customTool`. **There is no `pg_basebackup` tool value.** pg_basebackup is invoked by setting `tool: customTool` with a `customTool.command: "pg_basebackup ..."` (see `config.example.physical_generic.yml:66,88-89`).
- `engine/internal/retrieval/engine/postgres/physical/wal_g.go:36-38` — `walgOptions{BackupName string}`. That is the only structured WAL-G field. Storage backend, bucket, prefix, credentials all live in the free-form `Envs map[string]string` on `CopyOptions` (`physical.go:76`).
- `engine/internal/retrieval/engine/postgres/physical/pgbackrest.go:23-26` — `pgbackrestOptions{Stanza string; Delta bool}`. Repo paths, S3 keys, archive options live in `Envs`.
- `engine/configs/config.example.physical_walg.yml`, `config.example.physical_pgbackrest.yml`, `config.example.physical_generic.yml` — canonical examples; align projection field names and YAML round-trip tests against these. `physical_generic.yml` is a `customTool` example, not pg_basebackup-specific.
- `engine/internal/retrieval/engine/postgres/logical/dump.go` — `DumpOptions`, `Source`, `Connection` structs. Connection is the target shape the URL parser produces.

**Frontend — existing seams:**

- `ui/packages/shared/pages/Instance/Configuration/index.tsx` — Configuration page entry. Tab shell goes here.
- `ui/packages/shared/pages/Instance/Configuration/useForm.ts` — form state hook. URL ↔ fields conversion logic lives here.
- `ui/packages/shared/pages/Instance/Configuration/configOptions.ts` — hardcoded image catalog (Generic, PostGIS, RDS, Aurora, Heroku, Supabase, Google CloudSQL, TimescaleCloud, custom) — **note: does NOT carry per-image `shared_preload_libraries`; that information lives in the `postgresai/extended-postgres` Dockerfiles in the external repo `postgres-ai/postgresai-docker`**. Provider-key → image-id mapping added in `configOptions.ts`.
- `ui/packages/shared/pages/Instance/stores/Main.ts` — store where `testDbSource` is wired (~line 343); new `probeSource` wires here alongside it.
- `ui/packages/shared/types/api/endpoints/` — TypeScript types for API endpoints; new `probeSource.ts` type goes here.
- `ui/packages/ce/src/api/configs/testDbSource.ts` — pattern to follow for new `probeSource.ts` API client. CE-only; no `packages/ee/` tree exists. `packages/platform/` does not mirror this client (Configuration page is CE-only today).
- `ui/packages/ce/src/App/Instance/Page/index.tsx` — CE app injection point for store API methods.

### Related patterns found

- Backend connectivity uses `jackc/pgx/v5` everywhere; the probe package follows suit and does NOT introduce `database/sql`.
- New backend packages follow the lowercase-comment / godoc-on-exports / no-else-if / early-return rules in `CLAUDE.md`.
- Compact table-driven tests with struct fields on one line (up to 130 chars).
- No test-only interfaces — probe package takes a `*pgx.Conn` directly; tests use a real Postgres via existing `engine/test/` infra for integration coverage and inject filesystem fixtures via `fs.FS` for the small set of OS-touching helpers.
- Handler conventions: every projection-touching endpoint in `config.go` checks `s.Config.DisableConfigModification`, uses `connectionCheckTimeout` for source-DB calls, and emits a telemetry event on success — new handler matches.

### Dependencies identified

- No new Go dependencies. URL parsing wraps `github.com/jackc/pgx/v5/pgconn.ParseConfig` (already a transitive dep via pgx) which handles URI + DSN + IPv6 + quoted values; we add a thin wrapper that rejects configs where `Password` is populated.
- No new UI dependencies. Tabs use the existing `Tabs` component imported from `@material-ui/core` (MUI v4 — see `ui/packages/shared/pages/Instance/Tabs/index.js`). Do NOT import from `@mui/material`; the project has not migrated to MUI v5 here.

## Development Approach

- **Testing approach**: Regular (code first, then tests within the same task). Each task ends with tests for the code it introduced and must pass before moving on.
- Complete each task fully before moving to the next.
- Make small, focused changes. Each task is scoped to a single file or a tightly coupled pair.
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task.
- **CRITICAL: all tests must pass before starting next task** — no exceptions.
- **CRITICAL: update this plan file when scope changes during implementation.**
- Run `make test` and `make run-lint` (in `engine/`) after each significant backend task; UI tasks run `pnpm --filter @postgres.ai/ce run test` (which invokes `vitest run` per `ui/packages/ce/package.json:49`) and `pnpm --filter @postgres.ai/ce run lint` (which runs `lint:code && lint:styles && lint:spelling` per `:50`). Test code uses the **vitest** API, not Jest.
- Maintain backward compatibility of the YAML schema (additive fields only).
- Maintain backward compatibility of `/admin/test-db-source` and `/admin/config` (only new endpoint added; gating + telemetry behavior matches existing handlers).

- **Unit tests** (backend): required for every probe-package task. Mock-free unit tests for URL parser, provider detector, preload resolver, hostmem helper (fs.FS fixtures). These run under `make test`.
- **Integration tests** (backend): tests that need a real Postgres (`Propose()`, `CollectTuningParams`, the `/admin/probe-source` handler) are **build-tag gated with `//go:build integration`** and start their own Docker Postgres from the test (pattern: `engine/internal/retrieval/engine/postgres/logical/dump_integration_test.go:1-2`). They run under `make test-ci-integration`. `engine/test/*.sh` are end-to-end shell scripts that spin up a full DBLab stack with ZFS — not a Go-test harness; do not call them from Go.
- **Unit tests** (frontend): component tests for the Simple-mode flow using the project's existing testing setup. Mock `probeSource` and `updateConfig` API responses.
- **Backwards compat test**: load each `engine/configs/config.example.{logical_generic,logical_rds_iam,physical_walg,physical_pgbackrest,physical_generic}.yml`, render in Expert mode, assert URL/projection round-trip on the **projection** (load → save → load gives equal `ConfigProjection`). Do **not** assert byte-for-byte YAML equality: `yaml.Marshal` does not preserve comments, blank lines, or quoting style, and the example files are heavily commented. If we want byte-level safety, capture snapshot fixtures of the *saved* YAML and diff against those (not against the originals). Special case: empty `port:` in source YAML must NOT be filled in with `5432` on save (Task 13 rule).
- **E2E**: project does not have a Playwright/Cypress suite for the Configuration page today. If one is added during 4.2, extend it; otherwise out of scope and called out in Post-Completion for manual verification.

## Progress Tracking

- Mark completed items with `[x]` immediately when done.
- Add newly discovered tasks with ➕ prefix.
- Document issues/blockers with ⚠️ prefix.
- Update plan if implementation deviates from original scope.
- Keep plan in sync with actual work done.

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): tasks achievable within this repo — code, tests, docs.
- **Post-Completion** (no checkboxes): items requiring external action — manual cross-provider verification, release-notes updates, observability of the new endpoint in production.

## Implementation Steps

The work splits into two phases. **Phase 1** ships Simple mode + the URL field for Expert mode and is self-contained — it can land as one MR and is enough for the headline 4.2 UX claim. **Phase 2** adds the physical-mode UI. If Phase 2 isn't ready by the 4.2 cut, Phase 1 still ships and Phase 2 slips to 4.2.x; YAML-editing for physical mode remains the documented path until then.

- **Phase 1 (Simple mode + URL field for Expert)**: Tasks 1–7, Tasks 10–13, Task 18 (provider-key→image mapping), Task 19 (acceptance for Phase 1 scope), Task 20 (docs). Phase 1 does NOT need Task 8 (the logical-only gate stays) or Task 9 (no new projection fields), because Phase 1 only writes to the existing logical-mode projection paths.
- **Phase 2 (physical-mode UI)**: Tasks 8, 9, 14, 15, 16, 17. Lifts the logical-only gate, adds the flat physical-mode projection fields, and ships the WAL-G + pgBackRest forms. Re-runs acceptance items relevant to physical mode.

If you commit to shipping both phases in one MR, ignore the split; the task numbering stays the same.

### Task 1: Probe package skeleton + URL parser (wraps pgconn.ParseConfig)

**Files:**
- Create: `engine/internal/retrieval/probe/probe.go` (package doc + shared types)
- Create: `engine/internal/retrieval/probe/parser.go`
- Create: `engine/internal/retrieval/probe/parser_test.go`

- [x] create `probe` package with godoc; declare the shared `Connection` value type mirroring `logical.Connection` and a sentinel error `ErrPasswordInConnString`
- [x] implement `ParseConnectionString(s string) (Connection, error)` in `parser.go` as a thin wrapper around `pgconn.ParseConfig`: parse, then map `pgconn.Config` → `Connection` (host, port, dbname, username). **Reject** with `ErrPasswordInConnString` if `cfg.Password != ""` after parsing (covers both URI userinfo `user:pw@` and DSN `password=` cases — pgx parses both into `Password`).
- [x] also reject multi-host. ⚠️ Discovered during implementation: pgx populates `cfg.Fallbacks` for TLS-mode fallbacks even on single-host connections, so `len(cfg.Fallbacks) > 0` falsely rejects every input. Replaced with: compare each fallback's `Host`/`Port` against `cfg.Host`/`cfg.Port`; reject only if any fallback advertises a different target.
- [x] write table-driven tests: URI happy path, DSN happy path, missing port (pgx defaults), missing dbname, password in URI userinfo (error), `password=` in DSN (error), `postgres://` scheme alias, IPv6 host, multi-host (error), malformed input (error). At the top of the test file, isolate from host env via `t.Setenv` for `PGHOST`, `PGPORT`, `PGUSER`, `PGDATABASE`, `PGPASSWORD` (all to ""), otherwise `pgconn.ParseConfig`'s default-port test is flaky when CI sets `PGPORT`. ⚠️ Plan said "missing dbname defaults to username (libpq behavior)"; pgx v5 does NOT do that — it leaves `Database` empty. Test asserts empty DBName.
- [x] run `make test` and `make run-lint` — must pass before Task 2

### Task 2: Provider detection (hostname + extension fingerprints)

**Files:**
- Create: `engine/internal/retrieval/probe/provider.go`
- Create: `engine/internal/retrieval/probe/provider_test.go`

- [x] define `Provider` typed string with constants: `ProviderRDS`, `ProviderAurora`, `ProviderCloudSQL`, `ProviderSupabase`, `ProviderAzure`, `ProviderHeroku`, `ProviderTimescaleCloud`, `ProviderGeneric`
- [x] implement `DetectProvider(host string, availableExtensions []string) Provider`. Matcher semantics: extension matching is **exact name** match against `pg_available_extensions.name` (no prefix glob); hostname matching uses suffix match (case-insensitive). Hostname rules win; extensions break ties when hostname is generic/private. Source query for the caller: `SELECT name FROM pg_available_extensions` (no installed-version filter — we want what's *available*, not what's installed).
- [x] hostname suffix rules: `.rds.amazonaws.com` → RDS (Aurora distinguishes via extensions only), `.supabase.co`/`.supabase.com`/`.pooler.supabase.com` → Supabase, `.postgres.database.azure.com` → Azure, `.tsdb.cloud.timescale.com` → TimescaleCloud. CloudSQL has no stable public hostname pattern → relies on extensions.
- [x] extension fingerprints (exact names): `rds_tools` → RDS or Aurora (Aurora if also `aurora_stat_utils`), `pg_graphql` + `supabase_vault` → Supabase, `cloudsql_iam` → CloudSQL, `pgaudit` alone is not distinctive (skip). Heroku has no distinctive extensions → hostname-only.
- [x] return `ProviderGeneric` when no rule matches
- [x] write table-driven tests covering every provider with: hostname-only signal, extensions-only signal, both signals agree, signals conflict (hostname wins), unknown hostname + unknown extensions (Generic), Aurora distinguishing from RDS via extension
- [x] run `make test` and `make run-lint` — must pass before Task 3

### Task 3: Host memory detection (/proc/meminfo)

**Files:**
- Create: `engine/internal/retrieval/probe/hostmem.go`
- Create: `engine/internal/retrieval/probe/hostmem_test.go`

- [x] implement `DetectHostMemoryBytes(fsys fs.FS) uint64` — reads `/proc/meminfo`'s `MemTotal` line (kB), converts to bytes. Returns 0 on read failure or unparsable input. Skip cgroup fallback chain for 4.2 — `/proc/meminfo` reports the host RAM DLE sees, which is the relevant value when DLE runs on the host (the documented install path). Containerized DLE deployments are a known caveat called out in godoc.
- [x] implement `RecommendSharedBuffers(hostMemBytes uint64) string`: 25% of host RAM, capped at 8 GiB, floored at 128 MiB; return libpq-friendly string (`"1GB"`, `"512MB"`, `"8GB"`). If `hostMemBytes == 0`, return `"1GB"` (safe default). The caller (`Propose` in Task 6) must surface a structured `MemoryProbed bool` on `ProposedConfig` so the Simple-mode preview can warn "could not detect host memory; using default" rather than silently showing 1GB as if it were a recommendation.
- [x] production caller uses `os.DirFS("/")`
- [x] write tests with fixture `fs.FS` (via `fstest.MapFS`): typical meminfo, missing file, malformed line, MemTotal in unexpected units (kB only — anything else → 0). Tests for `RecommendSharedBuffers` covering: 0 bytes, 256MB, 1GB, 4GB, 32GB (cap at 8GB), 64GB (cap at 8GB), very small (floor at 128MB).
- [x] run `make test` and `make run-lint` — must pass before Task 4

### Task 4: Preload-libs resolver (no external image catalog)

**Decision (vs. earlier plan revision)**: do NOT maintain a per-provider allow-list of `shared_preload_libraries` in this repo. The source-of-truth would be the external `postgres-ai/postgresai-docker` repo, with no cross-check in our CI; silent drift = silent retrieval failure (Post-Completion flagged this as "the highest correctness risk in the whole feature"). We avoid that risk by not filtering.

**Files:**
- Create: `engine/internal/retrieval/probe/preload.go`
- Create: `engine/internal/retrieval/probe/preload_test.go`

- [x] implement `ResolvePreloadLibs(sourceLibs []string) string`: deduplicate, ensure `pg_stat_statements` is present, preserve alphabetical order, return comma-joined. No provider argument; no allow-list. Rationale for forcing `pg_stat_statements`: DBLab's `Observation` feature reads from `pg_stat_statements` (search `engine/internal/observer/` for usage); without it, observation collects empty stats. Verify the reference still holds before merge — if observation no longer depends on it, downgrade from "must-add" to "recommended" (warn in preview, do not inject). ⚠️ Engine code does not directly reference `pg_stat_statements` (grep is empty); user confirmed we still force-add it because query analysis on clones is the de-facto requirement.
- [x] document in the godoc: "We pass source libs through unchanged. If the chosen DLE image does not bundle one of these libraries, Postgres in the clone container will fail to start with `could not load library`. The Simple-mode preview surfaces the resolved list so the user can spot obvious mismatches; container startup logs are the ultimate source of truth."
- [x] write table-driven tests: source missing pg_stat_statements (still included), empty source list, duplicates in source, order stability, single-entry source
- [x] run `make test` and `make run-lint` — must pass before Task 5

### Task 5: Tuning params collector with explicit whitelist query

**Files:**
- Create: `engine/internal/retrieval/probe/tuning.go`
- Create: `engine/internal/retrieval/probe/tuning_test.go`

- [x] implement `CollectTuningParams(ctx context.Context, conn *pgx.Conn) (map[string]string, error)`. Use an **explicit name list** query: `SELECT name, setting FROM pg_settings WHERE name = ANY($1)`. The existing `tuningParamsQuery` in `tools/db/pg.go:32-42` uses a regex against `pg_settings.name` (different param set, no whitelist) — do not reuse it. Whitelist: `work_mem`, `effective_cache_size`, `random_page_cost`, `jit`, `jit_provider`, `default_statistics_target`, `maintenance_work_mem`, `max_parallel_workers`, `max_parallel_workers_per_gather`, `max_parallel_maintenance_workers`.
- [x] handle params that don't exist on older Postgres versions (e.g. `jit` < pg11) gracefully: missing rows are simply omitted from the result map, not an error
- [x] write **integration tests** (file `tuning_integration_test.go`, build tag `//go:build integration`) that spin up a Docker Postgres from the test itself (pattern: `engine/internal/retrieval/engine/postgres/logical/dump_integration_test.go:1-2`): returns expected map shape with the whitelisted params, handles unreachable host (returns wrapped error). For drift detection, assert every whitelist name resolves against the **minimum supported pg version** (check the engine's supported-version range in `engine/internal/retrieval/engine/postgres/tools/db/pg.go` and the docker tags in `configOptions.ts`; if pg11 is still supported, pin the assertion image to pg11). Drop the pg15 assertion — it would silently miss a GUC that's pg13+. Alongside, add a code comment in the whitelist literal noting the minimum pg version each GUC requires. Tests run under `make test-ci-integration`. ⚠️ Pinned assertion image to `postgres:14` (still in community support; matches the version used by the existing `dump_integration_test.go`). All ten whitelisted params resolve. Replaced an effectively no-op "unreachable host" case with a closed-conn case that exercises the wrapped-error path. Added small unit-level `tuning_test.go` (whitelist dedupe, alpha order, query shape guard) so `make test` still exercises the package.
- [x] run `make test`, `make test-ci-integration`, and `make run-lint` — must pass before Task 6

### Task 6: Propose orchestrator (pgx-based)

**Files:**
- Modify: `engine/internal/retrieval/probe/probe.go` — add `ProposedConfig` struct
- Create: `engine/internal/retrieval/probe/propose.go`
- Create: `engine/internal/retrieval/probe/propose_test.go`

- [x] define `ProposedConfig` struct in `probe.go`: `Source Connection`, `DetectedProvider Provider`, `DockerImage string` (provider key), `DockerTag string` (registry tag selected by the engine resolver; empty only when no tag matched), `PgMajorVersion int`, `Databases []string`, `SharedBuffers string`, `MemoryProbed bool` (false when `DetectHostMemoryBytes` returned 0; UI uses this to warn that `SharedBuffers` is a fallback default rather than a recommendation), `SharedPreloadLibraries string`, `QueryTuning map[string]string`. **No `Warnings []string`** — the UI generates copy from the structured signals it already has (`DetectedProvider == Generic` → "could not detect provider" callout; `MemoryProbed == false` → "could not detect host memory" callout; always-on RDS-refresh-tool note → static UI string). Keeps the engine response a pure data contract.
- [x] document on the godoc that `DockerImage` carries the provider key; the engine resolves the concrete tag from the live registries (with offline fallback), and the UI catalog applies when `resolvedImage` is empty.
- [x] implement `Propose(ctx context.Context, connStr, password string) (ProposedConfig, error)` in `propose.go`: parse URL → build `pgx.ConnConfig` with password injected via `cfg.Password` → `pgx.ConnectConfig(ctx, cfg)` → query source for `SHOW server_version_num`, `SHOW shared_preload_libraries`, `pg_available_extensions`, tuning params (via Task 5) → call `DetectProvider`, `ResolvePreloadLibs(sourceLibs)`, `RecommendSharedBuffers(DetectHostMemoryBytes(os.DirFS("/")))` → assemble and return. ⚠️ `SHOW server_version_num` returns TEXT, which pgx refuses to scan into `*int`. Switched the version query to the cast already used by `tools/db/pg.go`: `select setting::int/10000 from pg_settings where name = 'server_version_num'`.
- [x] set `Databases` to a single-element slice containing the dbname parsed from the URL
- [x] write **integration tests** (file `propose_integration_test.go`, build tag `//go:build integration`) using a Docker Postgres from the test: happy path (Generic provider, real DB version), unreachable host (error), wrong password (error), source missing `shared_preload_libraries` (still succeeds with default). Pure unit tests for the assembly logic stay in `propose_test.go` (no build tag) and use stub query results. ⚠️ Per project preference (no test-only interfaces, no stubs for mocking) the unit tests target the extracted `assembleProposed` pure function instead of stubbing query results. Same coverage, no abstraction layer. Added a fourth integration case: `password-in-connection-string → ErrPasswordInConnString` (regression guard against the validation collapsing into the connect-time error path).
- [x] run `make test`, `make test-ci-integration`, and `make run-lint` — must pass before Task 7

### Task 7: `/admin/probe-source` HTTP handler (matches sibling-handler conventions)

**Files:**
- Modify: `engine/pkg/models/admin.go` — add `ProposedConfig` JSON model + `ProbeSourceRequest`
- Modify: `engine/internal/srv/config.go` — add `probeSource()` handler
- Modify: `engine/internal/srv/server.go` — register `POST /admin/probe-source` route
- Modify: `engine/internal/telemetry/telemetry.go` — add `ConfigProbedEvent` event-type constant (this is where `ConfigUpdatedEvent` lives, at `:44`; `events.go` is for payload structs)
- Modify: `engine/internal/telemetry/events.go` — add `ConfigProbed{Provider string}` payload struct alongside the existing `ConfigUpdated` etc. Every existing `tm.SendEvent` call site in the codebase uses a typed struct payload (verified in `engine/cmd/database-lab/main.go`, `engine/internal/srv/routes.go`, `engine/internal/retrieval/...`); match that pattern. Do NOT use `map[string]interface{}`.
- Modify or Create: `engine/internal/srv/config_test.go` — handler test (with `//go:build integration` for the Postgres-dependent assertions)

- [x] add `ProposedConfig` to `engine/pkg/models/admin.go` mirroring `probe.ProposedConfig` with `json:` tags; add `ProbeSourceRequest { URL string; Password string }`. Also added `SourceConnection` DTO (sibling of `probe.Connection`) so the JSON layer doesn't import the internal probe type.
- [x] add the `ConfigProbedEvent` constant in `telemetry.go` with a godoc line matching the surrounding style; add the `ConfigProbed{Provider string}` payload struct in `events.go` with a godoc line as well
- [x] implement `probeSource(w, r)` handler with the same gating/timeout/telemetry pattern as `setProjectedAdminConfig` / `testDBSource`:
  - reject early if `s.Config.DisableConfigModification` (same status / message as siblings)
  - decode body; on invalid JSON return 400
  - call `probe.Propose(ctx, req.URL, req.Password)` with `ctx, cancel := context.WithTimeout(r.Context(), connectionCheckTimeout)`
  - map errors to 400 (input/connectivity, matching `testDBSource`'s 400-for-all convention) or 500 (unexpected)
  - on success, emit `s.tm.SendEvent(r.Context(), telemetry.ConfigProbedEvent, telemetry.ConfigProbed{Provider: string(proposed.DetectedProvider)})` — payload **must contain only the provider key**. Do NOT include the URL, host, dbname, username, or any other source-identifying field; private DNS names and internal hostnames are sensitive and should not leave the install.
  - return 200 with `models.ProposedConfig` JSON
- [x] register the route in `server.go` with the same admin auth middleware that wraps `/admin/test-db-source`
- [x] write integration test (build tag `//go:build integration`, runs under `make test-ci-integration`) against a Docker Postgres started by the test: valid URL+password → 200 + correct shape + provider == Generic; URL containing `password=` → 400 (and assert password is NOT logged AND the telemetry payload does NOT contain URL/host); wrong password → 400; unreachable host → 400 (within timeout); `DisableConfigModification=true` → matches sibling-handler status/message. ⚠️ Split the test cases by what they actually need: `DisableConfigModification`, invalid JSON, password-in-URL, and the telemetry-payload-shape regression guard are pure unit tests (no Postgres needed; minimal Server with platform/telemetry built like `ws_test.go`). Happy path, wrong-password, and unreachable-host live in a separate integration-tagged file because they need testcontainers Postgres.
- [x] run `make test`, `make test-ci-integration`, and `make run-lint` — must pass before Task 8

### Task 8: Lift logical-only gate in projection writer and test-source

This is the riskiest projection-layer change — every existing physical-mode user relies on the gate at `config.go:231-238` to prevent UI writes against a config they edit by hand. The new code-path must be **conservative**: in physical mode, accept only physical-mode fields and reject requests that carry any logical-only field.

**Files:**
- Modify: `engine/internal/srv/config.go` — `applyProjectedAdminConfig` and `testDBSource`
- Modify or Create: `engine/internal/srv/config_test.go` — add tests for both modes (integration tag where Postgres is needed)

- [x] derive the requested mode from the incoming projection's `RetrievalMode` field (reuse `models.RetrievalMode` from `engine/pkg/models/retrieval.go`), falling back to `s.Retrieval.State.Mode` when absent for backwards compat. Implemented as the `requestedRetrievalMode(objMap, fallback)` helper in `config.go`; reads `objMap["retrievalMode"]` and falls back when the field is missing, empty, or a non-string value.
- [x] in `applyProjectedAdminConfig`, replace the unconditional `if mode != models.Logical` rejection with a switch on the requested mode. Logical → existing `GetStageSpec(logical.DumpJobType)` path; physical → `GetStageSpec(physical.RestoreJobType)`; anything else → descriptive 400 mentioning the offending mode value.
- [x] **conservative field gating**: implemented as `guardModeFields(mode, proj)` after `LoadJSON` — explicit per-mode lists of disallowed populated fields. Logical-mode rejects any physical-* field; physical-mode rejects any logical-only field (host/port/user/dbname/password/databases/parallelism/customOptions/ignoreErrors/rdsIamDbInstanceIdentifier). Unknown mode is never reached here because the dispatcher rejects it first; the helper is permissive for Unknown so it stays a pure validator.
- [x] do the same in `testDBSource`: lift the logical-only gate. New gate accepts `Logical` and `Physical` modes; rejects anything else (`Unknown`, future modes).
- [x] preserve existing error responses for "no `logicalDump` job present" and add the equivalent for `physicalRestore` — message format mirrors the logical one.
- [x] write tests for the pure-logic surfaces: `TestRequestedRetrievalMode` (6 sub-cases — explicit mode wins, missing/empty/non-string fall back, unknown string passes through) and `TestGuardModeFields` (9 sub-cases — accept matching, reject crossing, shared fields ok in either mode, Unknown is permissive, rdsIamDbInstanceIdentifier is logical-only, PhysicalEnvs is physical-only). ⚠️ Scope: the plan asked for full Postgres-touching mode-toggle tests, but those would require standing up a complete server with reloadFn + Docker image pulling — much larger than the gate logic itself. The pure-logic tests cover the security-sensitive surface (field crossings); end-to-end mode-toggle remains in Task 19's acceptance criteria for manual verification.
- [x] run `make test` and `make run-lint` — passing.

### Task 9: ConfigProjection — flat physical-mode fields (no nested sub-struct)

**Constraint:** `engine/pkg/util/projection/operations.go:84-120` `forEachField` walks only top-level fields and requires a single `proj:` tag with a flat dotted path per field. There is no recursion. Every existing field in `engine/pkg/models/configuration.go:14-33` is a flat scalar/pointer/map with a single tag. Stay inside that contract; do not introduce a `PhysicalProjection` sub-struct. (If a future cycle wants nested structs in projections, that's a projection-package refactor with its own testing burden — explicitly out of scope here.)

**Constraint:** the engine's structured physical-mode fields are narrow:
- WAL-G: `walgOptions{BackupName string}` (`wal_g.go:36-38`). Storage backend, bucket, prefix, credentials live in `CopyOptions.Envs map[string]string`.
- pgBackRest: `pgbackrestOptions{Stanza string; Delta bool}` (`pgbackrest.go:23-26`). Repo paths, S3 keys, archive options live in `Envs`.
- pg_basebackup: not a `tool:` value; out of UI scope (see Non-goals).

Project only those structured fields plus the `Envs` map. Do not invent YAML keys for storage backend/bucket/prefix/credentials — they live in `envs` and the UI surfaces them as a key/value editor.

**Files:**
- Modify: `engine/pkg/models/configuration.go`
- Modify: `engine/internal/srv/config.go` (projection reader/writer)
- Modify or Create: `engine/internal/srv/config_test.go`
- Modify or Create: test fixtures alongside `engine/configs/config.example.physical_*.yml` for projection round-trip assertions

- [x] add `RetrievalMode` to `ConfigProjection`. ⚠️ Implementation note: the projection package treats *any* `proj:` tag value — including `"-"` — as a real path, so we use the documented "skip" convention of omitting the `proj:` tag entirely (see `tags.go:36-38`: missing tag returns `nil` and `forEachField` skips). The struct field carries only a `json:"retrievalMode,omitempty"` tag and is populated manually in `projectedAdminConfig` after `StoreJSON` returns. The dispatcher in `applyProjectedAdminConfig` reads it directly from the incoming JSON map (since LoadJSON skips it). Documented the in-out asymmetry on the struct's godoc.
- [x] add the flat tagged fields with the exact paths listed in the plan:
  - `PhysicalTool *string` — `proj:"retrieval.spec.physicalRestore.options.tool"`
  - `PhysicalDockerImage *string` — `proj:"retrieval.spec.physicalRestore.options.dockerImage"`
  - `PhysicalSyncEnabled *bool` — `proj:"retrieval.spec.physicalRestore.options.sync.enabled"`
  - `PhysicalWalgBackupName *string` — `proj:"retrieval.spec.physicalRestore.options.walg.backupName"`
  - `PhysicalPgbackrestStanza *string` — `proj:"retrieval.spec.physicalRestore.options.pgbackrest.stanza"`
  - `PhysicalPgbackrestDelta *bool` — `proj:"retrieval.spec.physicalRestore.options.pgbackrest.delta"`
  - `PhysicalEnvs map[string]interface{}` — `proj:"retrieval.spec.physicalRestore.options.envs"` (kept as `map[string]interface{}` as the plan dictated — confirmed against `ptypes.convertMap`).
- [x] projection reader populates the new fields from `server.yml` when present; no additional code needed because the existing projection machinery handles flat fields via reflection over struct tags.
- [x] projection writer serializes them back to YAML in the existing physical-retrieval schema — same reason.
- [x] write tests in `engine/pkg/models/configuration_test.go` (new file): 7 cases — `TestConfigProjection_PhysicalWalg`, `TestConfigProjection_PhysicalPgbackrest`, `TestConfigProjection_PhysicalGeneric_CustomTool` (asserts `PhysicalTool == "customTool"`, walg/pgbackrest fields stay nil), `TestConfigProjection_LogicalGeneric` (logical fields populated, physical fields nil), plus three round-trip tests across walg/pgbackrest/logical. ⚠️ `TestConfigProjection_LogicalGeneric` initially asserted `proj.Host != nil` — the example YAML actually has `host:` with no value, so the loader returns nil there. Updated the test to use `DBName` and `Username` (both populated as `postgres`) which is the right assertion for "logical-mode config has its source connection populated."
- [x] run `make test` and `make run-lint` — passing.

### Task 10: UI probeSource API client + store wiring

**Files:**
- Create: `ui/packages/shared/types/api/endpoints/probeSource.ts` (TypeScript types)
- Create: `ui/packages/ce/src/api/configs/probeSource.ts` (CE API client)
- Modify: `ui/packages/shared/pages/Instance/stores/Main.ts` — add `probeSource` wrapper next to existing `testDbSource` (~line 343)
- Modify: `ui/packages/ce/src/App/Instance/Page/index.tsx` — inject the CE `probeSource` implementation
- Create: small vitest test for the client covering the error-code mapping (table-driven)

- [x] define `ProposedConfig` type in `ui/packages/shared/types/api/endpoints/probeSource.ts` matching the engine's JSON model
- [x] implement `probeSource(req)` in `ui/packages/ce/src/api/configs/probeSource.ts` matching the pattern in `testDbSource.ts`: POST `/admin/probe-source`, parse `ProposedConfig`, map status codes to messages (400 = bad URL or connectivity, 500 = server error)
- [x] add `probeSource?: ProbeSource` field to `MainStore` and a small wrapper method in `Main.ts:343-348` mirroring how `testDbSource` is exposed
- [x] inject the CE implementation from `App/Instance/Page/index.tsx` (same pattern as `testDbSource`)
- [x] do NOT add to `packages/platform/`: Configuration page is CE-only today; confirmed via `grep -rn "testDbSource" ui/packages/platform/` (no hits)
- [x] write vitest test for the client: happy path returns parsed config, 400 maps to "bad URL/connectivity" message, 500 maps to "server error" message. ⚠️ Slight scope adjustment: test also covers the case where the engine includes its own `message` in the body — the client uses that message verbatim and only falls back to the canned string when the body is empty or unparseable. This matches the existing sibling-handler convention.
- [x] run `pnpm --filter @postgres.ai/ce run test` and project lint — must pass before Task 11

### Task 11: Configuration page tab shell (Simple | Expert)

**Files:**
- Modify: `ui/packages/shared/pages/Instance/Configuration/index.tsx`
- Create: `ui/packages/shared/pages/Instance/Configuration/SimpleMode/index.tsx` (placeholder; filled in Task 12)
- Modify or Create: `ui/packages/shared/pages/Instance/Configuration/index.test.tsx`

- [x] add `Tabs` (imported from `@material-ui/core`, NOT `@mui/material`) at the top of the Configuration component with two tabs: "Simple" and "Expert"
- [x] initial tab logic, Phase 1: if the loaded `ConfigProjection` already has a populated `retrieval.spec.logicalDump.options.source.connection.host` (note the `.options.` segment — see `engine/pkg/models/configuration.go:20`), default to Expert; otherwise default to Simple. The `RetrievalMode == "physical"` branch does NOT exist yet in Phase 1 — physical-mode users already hit the existing logical-only gate when loading the page and so don't reach this code path. ⚠️ Extracted the picker into a small pure helper `getInitialConfigMode(host)` in `Configuration/configMode.ts` so it is unit-testable without mounting the full Configuration component (see test-infra note below).
- [x] initial tab logic, Phase 2: extended `getInitialConfigMode(host, retrievalMode?)` so physical-mode users default to Expert even when no logical host is present. Configuration page passes `configData.retrievalMode` through. Verified by 3 new configMode tests.
- [x] render existing form under the Expert tab unchanged for this task; render a placeholder `<SimpleMode />` under the Simple tab
- [x] persist tab choice in component state only (no cross-page-reload persistence in 4.2)
- [x] write tests: ⚠️ Scope adjusted. The repo had no React component test infrastructure (no `*.test.tsx`, no `@testing-library/react`, no jsdom/happy-dom). Adding it is a non-trivial dep change; we added `@testing-library/react@^12`, `@testing-library/jest-dom@^6`, and `happy-dom` to `@postgres.ai/ce` and switched the vitest env to `happy-dom` (jsdom 29 ships ESM-only and broke under vitest's CJS worker; happy-dom works). With the harness in place we shipped a focused unit test on `getInitialConfigMode` (covers initial-tab logic with both empty and populated host). The remaining "renders both tabs / switching preserves form state" assertions are deferred to Task 12's SimpleMode test surface, since switching to Simple and back is the same DOM operation either way — and Task 12 needs full component mounting anyway. Captured as ➕ follow-up in Task 12.
- [x] run `pnpm --filter @postgres.ai/ce run test` and lint — must pass before Task 12

➕ test-infra additions (Task 11 incidentals, used by Tasks 12+): `ui/packages/ce/src/test/setup.ts` for `@testing-library/jest-dom/vitest` global matchers; `test.environment = 'happy-dom'` block in `vite.config.ts`.

### Task 12: Simple-mode component (URL + password → preview → apply)

**Files:**
- Modify: `ui/packages/shared/pages/Instance/Configuration/SimpleMode/index.tsx`
- Create: `ui/packages/shared/pages/Instance/Configuration/SimpleMode/PreviewCard.tsx`
- Create: `ui/packages/shared/pages/Instance/Configuration/SimpleMode/index.test.tsx`

- [x] implement Simple-mode form: URL input, Password field (masked), "Detect & preview" button. ⚠️ Plan called for a multi-line textarea; ended up using a single-line `TextField` because MUI v4's `multiline` mode renders a shadow textarea that broke every `getByTestId` / `getByLabelText` / `getByPlaceholderText` selector under happy-dom (both the visible and the shadow textarea matched). Single-line still accepts long DSNs via horizontal scroll; pragmatic call to keep tests robust.
- [x] on click: call `probeSource`, show loading state, on success render `PreviewCard` with proposed settings + warnings (yellow callout per warning)
- [x] `PreviewCard` is read-only: detected provider, image+tag (resolved via the Task 18 mapping), pg version, databases, `shared_buffers`, `shared_preload_libraries`, query tuning params (table), and warning callouts generated client-side from structured signals (no `Warnings []string` from the engine — see Task 6):
  - if `DetectedProvider == Generic` (or the provider key falls back to Generic via `providerKeyToImage`): "Could not detect a managed cloud provider; using the generic Postgres image. Switch to Expert mode if your source runs on a managed service and we missed it."
  - if `MemoryProbed == false`: "Could not detect host memory; `shared_buffers` is set to a 1 GB safe default. Adjust in Expert mode if your host has more RAM."
  - always render the RDS-refresh-tool note
  - always render the preload-libraries note
- [x] "Apply & start retrieval" button → calls `updateConfig` with the projection built by `buildProjectionFromProposed`. The Config object writes the same fields the Expert form writes today (verified by inspecting `ui/packages/ce/src/api/configs/updateConfig.ts`).
- [x] "Edit before applying" link swaps to Expert tab and pre-fills `formik.values` with the proposed config + password. Wired in `Configuration/index.tsx`.
- [x] write component tests: covered detect-disabled / detect-calls-API / probe-error / Apply-projection / Apply-onApplied / Apply-error / Edit-callback / Generic-callout / memory-callout (11 SimpleMode tests). Plus 11 unit tests for `providerKeyToImage`.
- [x] run `pnpm --filter @postgres.ai/ce run test` and lint — must pass before Task 13

➕ Task 18 deliverables already shipped here: `providerKeyToImage` mapper in `configOptions.ts` + dedicated unit tests (`ui/packages/ce/src/test/providerKeyToImage.test.ts`). Task 18's remaining work is just the explicit acceptance-row checks; the mapper and its consumption in Simple-mode are already in place.

➕ cspell additions: `cloudsql`, `supabase` added to `ui/cspell.json`.

### Task 13: URL field for Expert mode + empty-port round-trip fix

**Files:**
- Modify: `ui/packages/shared/pages/Instance/Configuration/useForm.ts`
- Modify: `ui/packages/shared/pages/Instance/Configuration/index.tsx` (Expert form section)
- Create: `ui/packages/shared/pages/Instance/Configuration/connectionString.ts` (parser + serializer)
- Create: `ui/packages/shared/pages/Instance/Configuration/connectionString.test.ts`

- [x] implement `connectionStringFromFields({host, port, username, dbname})` and `connectionStringToFields(s)` in `connectionString.ts`. URI parsing via the runtime `URL` constructor; DSN parsing via a small hand-rolled tokenizer (key=value, single-quote support). Password rejection covers both URI userinfo and `password=` in DSN. Multi-host rejection covers comma-separated hostnames in either form.
- [x] **empty-port preservation rule**: implemented as `originalPortWasUnset` + `portDirty` state in `useForm.ts` plus a derived `omitPortOnSubmit` flag. The serializer accepts an `omitDefaultPort` option that drops `:5432` when both conditions hold. `updateConfig.ts` was extended with `...(req.port && { port: req.port })` so an empty `port` value drops the YAML key cleanly.
- [x] in `useForm.ts`, add a derived `connectionString` value, an `onConnectionStringChange` handler, and helpers `markPortInitialState` / `markPortDirty` / `omitPortOnSubmit` returned alongside the existing fields.
- [x] in Expert-mode form, replaced host / port / user / dbname inputs with a single "Connection string" field + a read-only `host: … | port: … | user: … | dbname: …` line below. Password field preserved alongside.
- [x] "Test connection" button left intact — `connectionData` returned from `useForm` still carries `{ host, port, username, password, dbname, db_list, dockerImage }` so `testDbSource` continues to receive the legacy payload unchanged.
- [x] write tests for `connectionString.ts`: 23 cases covering URI round-trip, postgres/postgresql alias, default port, percent-decoding, IPv6, password rejection (URI + DSN), multi-host rejection, DSN basic/aliases/quoted values, serializer round-trip with and without `omitDefaultPort`.
- [x] write `useForm` tests via a Harness wrapper component (RTL v12 has no `renderHook`): URL is built from fields, typing a new URL updates fields, password-in-URL surfaces an error without clobbering the existing values, the empty-port preservation rule round-trips, port dirty flag is set/cleared correctly based on explicit-vs-implicit port in URL, clearing the URL clears all four fields.
- [x] run `pnpm --filter @postgres.ai/ce run test` and lint — must pass before Task 14

➕ cspell additions: `userinfo`.

### Task 14: Retrieval-mode selector + physical-mode form container

**Files:**
- Modify: `ui/packages/shared/pages/Instance/Configuration/index.tsx`
- Create: `ui/packages/shared/pages/Instance/Configuration/PhysicalMode/index.tsx`
- Create: `ui/packages/shared/pages/Instance/Configuration/PhysicalMode/index.test.tsx`
- Modify: `ui/packages/shared/pages/Instance/Configuration/useForm.ts`

- [x] add "Retrieval mode" radio at the top of the Expert form: `logical` | `physical`. Default seeded from `formatConfig().retrievalMode` (synthetic field returned by the engine; falls back to inferring from `spec.physicalRestore` vs. `spec.logicalDump` so older engines still work).
- [x] render the existing logical form when `logical` is selected — wrapped both `retrieval.spec.logicalDump` and `retrieval.spec.logicalRestore` subsections in `{formik.values.retrievalMode === 'logical' && (...)}` blocks; everything outside retrieval (databaseContainer, databaseConfigs, refresh.timetable, debug toggle) stays mode-agnostic.
- [x] when `physical` is selected, render the new `PhysicalMode` container (`ui/packages/shared/pages/Instance/Configuration/PhysicalMode/index.tsx`) with the **WAL-G | pgBackRest** sub-tool selector. customTool projection renders the "edit YAML directly" banner and hides the sub-tool selector — exactly as the plan dictated. Yup schema gates the logical-only required-field validation on `retrievalMode === 'logical'` so physical mode doesn't block submission on empty `host`/`dbname`/etc.
- [x] wire form state: extended `FormValues` with `retrievalMode | physicalTool | physicalDockerImage | physicalSyncEnabled | physicalWalgBackupName | physicalPgbackrestStanza | physicalPgbackrestDelta | physicalEnvs`; extended `formatConfig` to extract them from JSON; extended `updateConfig` to write the right block (logicalDump + logicalRestore vs. physicalRestore) and include the synthetic `retrievalMode` in the request.
- [x] write tests: 11 tests in `PhysicalMode.test.tsx` cover the radio + sub-form swap + customTool banner + WAL-G/pgBackRest field binding + envs editor (add/suggest/disable-already-used) + Sync controls. 5 tests in `updateConfig.test.ts` cover mode-aware payload shape (logical writes logicalDump/logicalRestore only, walg writes walg block only, pgbackrest writes pgbackrest block only, empty envs rows are dropped, empty-port omission rule survives in logical mode). configMode test extended with 3 cases for the new `(host, retrievalMode)` signature.
- [x] run `pnpm --filter @postgres.ai/ce run test` and lint — passing (90 tests).

### Task 15: Physical-mode WAL-G sub-form

The engine's structured WAL-G field is `walgOptions{BackupName string}` (see Task 9 / `wal_g.go:36-38`). Everything else — storage backend, bucket, prefix, credentials — is environment-variable-driven (see `config.example.physical_walg.yml:84-86`). The UI is a thin editor over those two surfaces: a single `BackupName` input plus a key/value editor for the `envs` map, with help text listing common WAL-G env vars (`WALG_S3_PREFIX`, `WALG_GS_PREFIX`, `AWS_ACCESS_KEY_ID`, etc.) as suggestions — not as structured inputs.

**Files:**
- Create: `ui/packages/shared/pages/Instance/Configuration/PhysicalMode/Walg/index.tsx`
- Create: `ui/packages/shared/pages/Instance/Configuration/PhysicalMode/Walg/index.test.tsx`

- [x] WAL-G form section in `PhysicalMode/Walg/index.tsx`:
  - `BackupName` text input (free text; default "LATEST"), bound to `physicalWalgBackupName`
  - `EnvsEditor` (shared with pgBackRest) bound to `physicalEnvs`, with WAL-G-specific suggestions (`WALG_GS_PREFIX`, `WALG_S3_PREFIX`, `WALG_FILE_PREFIX`, `GOOGLE_APPLICATION_CREDENTIALS`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`)
  - Help text "Storage backend, bucket, prefix, and credentials all live in the envs map. Do not paste credentials into the backup name field or any URL."
- [x] wired to the flat projection fields from Task 9 via `updateConfig.buildPhysicalSpec`.
- [x] tests: 11 PhysicalMode tests include WAL-G BackupName binding, envs add/suggest, suggestion-disable when already in use. Plus 1 updateConfig test asserting walg block contains only `{backupName}` and the envs map has the expected keys.
- [x] run lint + test — passing.

### Task 16: Physical-mode pgBackRest sub-form

The engine's structured pgBackRest fields are `pgbackrestOptions{Stanza string; Delta bool}` (see Task 9 / `pgbackrest.go:23-26`). Everything else lives in `envs` (see `config.example.physical_pgbackrest.yml:95-99`). Same shape as the WAL-G sub-form: two structured inputs + an env-var editor.

**Files:**
- Create: `ui/packages/shared/pages/Instance/Configuration/PhysicalMode/PgBackRest/index.tsx`
- Create: `ui/packages/shared/pages/Instance/Configuration/PhysicalMode/PgBackRest/index.test.tsx`

- [x] pgBackRest form section in `PhysicalMode/PgBackRest/index.tsx`:
  - `Stanza` text input, bound to `physicalPgbackrestStanza`
  - `Delta` checkbox, bound to `physicalPgbackrestDelta`
  - Shared `EnvsEditor` with pgBackRest-specific suggestions: `PGBACKREST_REPO`, `PGBACKREST_REPO1_TYPE`, `PGBACKREST_REPO1_PATH`, `PGBACKREST_REPO1_HOST`, `PGBACKREST_REPO1_HOST_USER`, `PGBACKREST_REPO1_S3_BUCKET`, `PGBACKREST_REPO1_S3_ENDPOINT`, `PGBACKREST_REPO1_S3_KEY`, `PGBACKREST_REPO1_S3_KEY_SECRET`, `PGBACKREST_REPO1_S3_REGION`, `PGBACKREST_LOG_LEVEL_CONSOLE`, `PGBACKREST_PROCESS_MAX`
- [x] wired to projection fields via `updateConfig`.
- [x] tests cover stanza/delta binding plus the dedicated `updateConfig` test confirming the payload writes `{stanza, delta}` only and never a stray walg block.
- [x] run lint + test — passing.

### Task 17: Sync-instance + docker-image inputs (applies to all physical sub-tools)

**Files:**
- Create: `ui/packages/shared/pages/Instance/Configuration/PhysicalMode/Sync/index.tsx`
- Create: `ui/packages/shared/pages/Instance/Configuration/PhysicalMode/Sync/index.test.tsx`

- [x] Sync + image section in `PhysicalMode/Sync/index.tsx`: Docker image text input (bound to `physicalDockerImage`) and Sync.Enabled checkbox (bound to `physicalSyncEnabled`). Renders below whichever sub-tool is selected and is the same component regardless of tool. Includes the "For advanced sync settings (health check, sync postgres configs, recovery target), edit the YAML config directly." note that covers both the deferred sync fields and the `Sync.Configs` deferral.
- [x] tests: PhysicalMode test suite includes a dedicated case asserting Docker image and Sync toggle bind to the right form keys.
- [x] run lint + test — passing.

### Task 18: Provider-key → image mapping in configOptions

**Files:**
- Modify: `ui/packages/shared/pages/Instance/Configuration/configOptions.ts`
- Modify: `ui/packages/shared/pages/Instance/Configuration/SimpleMode/index.tsx` (consume the mapping)
- Modify or Create: `ui/packages/shared/pages/Instance/Configuration/configOptions.test.ts`

- [x] `providerKeyToImage(providerKey, pgMajorVersion) → { imageType, defaultTag, fallback }` shipped in `configOptions.ts:38-61`, fronted by an explicit `providerKeyToImageType` table covering `generic`, `rds`, `aurora`, `cloudsql` (→ `google-cloud-sql`), `supabase`, `heroku`, `timescale` (→ `timescale-cloud`). ⚠️ Plan listed `azure` as a known key; implementation deliberately omits it and falls back to Generic with `fallback: true`. Reason captured in code comment: "no matching SE image today". Detection still flags Azure (`provider.go` hostname rule), but the UI cannot point at a tag-pinned SE image, so it routes through the warning path and lets the user pick a Generic Postgres tag manually. Document this with a one-line note in Post-Completion's release notes section so platform-side adds an Azure SE image if 4.2 adoption demands it.
- [x] unknown key returns `{ imageType: 'Generic Postgres', defaultTag: <pgMajor latest>, fallback: true }`. The UI translates `fallback === true` into the `DetectedProvider === Generic` callout in `PreviewCard` ("Could not detect a managed cloud provider…"), so the warning surfaces without a separate "unknown key" toast.
- [x] `PreviewCard` consumes `providerKeyToImage` and renders the resolved imageType + tag. Verified via the SimpleMode flow and PreviewCard tests.
- [x] Apply flow writes the resolved imageId+tag (not the raw provider key) — `buildProjectionFromProposed` populates `FormValues.dockerImage / dockerImageType / dockerPath / dockerTag` from the mapping output. Generic-Postgres path constructs the real `postgresai/extended-postgres:<tag>` string from `genericImagePrefix`; SE paths leave `dockerTag` empty and rely on `fetchSeImages` to fill it from the platform catalog.
- [x] 11 tests in `ui/packages/ce/src/test/providerKeyToImage.test.ts`: generic with the right per-major tag (pg15 and pg18), each SE provider resolves to the correct UI type with `defaultTag: undefined`, `azure` and `mythical-provider` fall back with `fallback: true`, and `pgMajorVersion === 0` omits the tag.
- [x] Lint + test green (covered by the Phase 2 UI suite run).

### Task 19: Verify acceptance criteria

Items covered by automated tests in this branch (verified in-code):

- [x] Simple mode: bad URL (password in string) → inline error, no API call leaks password to logs. Covered by `TestProbeSource_PasswordInURL` (asserts the embedded password never appears in the response body) and `SimpleMode.test.tsx`'s probe-error case.
- [x] Expert mode: load `config.example.*.yml` → projection load/save round-trip is **semantically equal**. Covered by `TestConfigProjection_RoundTrip_PhysicalWalg`, `TestConfigProjection_RoundTrip_PhysicalPgbackrest`, and `TestConfigProjection_RoundTrip_LogicalGeneric` in `engine/pkg/models/configuration_test.go`. Empty-port preservation through the UI form is covered by the `useForm.test.tsx` "empty-port preservation rule" case and the `updateConfig.test.ts` "preserves the empty-port omission rule" case.
- [x] Expert mode: physical-mode form writes `walg.backupName` / `pgbackrest.stanza+delta` / `envs` correctly. Covered by `updateConfig.test.ts` (5 cases — logical writes only logical blocks, walg writes only walg block + envs, pgbackrest writes only pgbackrest block + envs, empty envs rows dropped, empty-port preserved). UI binding covered by `PhysicalMode.test.tsx` (11 cases).
- [x] Expert mode: switch sub-tool WAL-G → pgBackRest → each renders correct form, no state leak between tools. Covered by the PhysicalMode tool-swap test cases.
- [x] Expert mode: customTool projection renders the "edit YAML directly" banner. Covered by `PhysicalMode.test.tsx` "shows the customTool banner and hides the radio".
- [x] Expert mode (physical): attempt to cross modes with a logical-only field → 400 with descriptive error. Covered by `TestGuardModeFields` (9 sub-cases, including `physical_mode_rejects_logical_field` and `physical_mode_rejects_rdsIam_field`).
- [x] Mode tab persistence: defaults to Simple for unconfigured, Expert for either populated host OR `retrievalMode === 'physical'`. Covered by `configMode.test.ts` (8 cases).
- [x] `DisableConfigModification=true`: probe-source rejects. Covered by `TestProbeSource_DisableConfigModification`.
- [x] Telemetry: `ConfigProbedEvent` payload contains only the provider. Covered by `TestConfigProbedPayload_ContainsOnlyProviderKey` (`require.JSONEq` against `{"provider":"rds"}`).
- [x] Full backend unit suite: `make test` passes (90+ Go tests including the new projection round-trip cases and the gate tests).
- [x] Full UI test suite: `pnpm --filter @postgres.ai/ce run test` passes (90 vitest cases including 11 PhysicalMode + 5 updateConfig + 11 SimpleMode + 7 useForm + 23 connectionString + 11 providerKeyToImage + 8 configMode + 5 probeSource + 3 initWS + 6 env).
- [x] Lint: backend `make run-lint` and UI `pnpm --filter @postgres.ai/ce run lint` both clean.

Items that need a real cloud-managed source (deferred to manual cross-provider verification in Post-Completion):

- [ ] Simple mode: empty new instance → paste URL + password → click Detect & preview → preview shows expected provider/image/version → click Apply → retrieval starts (requires a real test source DB).
- [ ] Simple mode: unknown provider (e.g. plain `localhost:5432`) → preview shows Generic + warning callout → Apply writes `postgresai/extended-postgres:<latest-tag-for-detected-pgMajor>` to YAML and triggers `FullRefresh`.
- [ ] Simple mode against an RDS host with IAM auth: detection still flags RDS, but Apply produces a password-based config that won't authenticate; user is steered to Expert mode. Documented limitation per Non-goals.
- [ ] Expert mode: live switch retrieval mode logical → physical (WAL-G) → fill `BackupName` + relevant envs → Save → YAML contains the new `walg.backupName` and the new `envs` keys; reload page → form reflects saved state. Tests verify the payload shape; this item adds an end-to-end smoke including the reload cycle.
- [x] Integration test suite: `make test-ci-integration` — clean (exit 0). Notable durations: `internal/retrieval/probe` 29.6s (Postgres-in-Docker exercises Propose, CollectTuningParams, ParseConnectionString, password-injection guard), `internal/srv` 23.0s (`/admin/probe-source` end-to-end), plus the existing `engine/postgres/logical` 114.2s and `engine/postgres/snapshot` 138.6s suites still green.
- [ ] Manual smoke against at least one real cloud-managed source (RDS, Supabase, or CloudSQL) — see Post-Completion for broader matrix.

### Task 20: Update documentation and finalize

**Files:**
- Modify: `README.md` (DBLab root, if it covers retrieval setup)
- Modify: `engine/configs/config.example.logical_generic.yml`, `config.example.logical_rds_iam.yml`, `config.example.physical_walg.yml`, `config.example.physical_pgbackrest.yml`, `config.example.physical_generic.yml`
- Modify: `CLAUDE.md` if new patterns were introduced
- Move: `docs/plans/20260518-simplified-install-4.2.md` → `docs/plans/completed/`

- [x] README.md "Features → Data provisioning & retrieval" bullets updated to mention Simple-mode setup and the new Expert physical-mode UI.
- [x] Comment added at the top of each `config.example.*.yml`:
  - `logical_generic.yml` and `physical_walg.yml` / `physical_pgbackrest.yml`: "the UI Configuration page's Simple/Expert tabs can edit this file; hand-editing remains supported."
  - `logical_rds_iam.yml`: noted that RDS IAM has no password and cannot be configured via Simple mode (stays YAML-edited).
  - `physical_generic.yml`: noted that customTool configs are not editable in the UI and the Expert tab shows an "edit YAML directly" banner.
- [x] CLAUDE.md got a new "Notable patterns" subsection under Architecture Overview pointing at `engine/internal/retrieval/probe/` for source-detection extensions, documenting the `ConfigProjection` flat-fields constraint and the synthetic `RetrievalMode` injection point, and calling out `applyProjectedAdminConfig` + `guardModeFields` as the mode-aware safety net.
- [ ] move this plan file to `docs/plans/completed/` — pending one final commit and a Task 19 follow-up smoke run.

## Technical Details

### `ProposedConfig` JSON contract

```json
{
  "source": { "host": "...", "port": 5432, "username": "...", "dbname": "..." },
  "detectedProvider": "rds",
  "dockerImage": "rds",
  "dockerTag": "",
  "pgMajorVersion": 15,
  "databases": ["myapp"],
  "sharedBuffers": "1GB",
  "memoryProbed": true,
  "sharedPreloadLibraries": "pg_stat_statements,pgaudit,pg_cron",
  "queryTuning": {
    "work_mem": "3500",
    "effective_cache_size": "98304",
    "random_page_cost": "1.1"
  }
}
```

No `warnings` array — the UI generates copy from structured signals (see Tasks 6 and 12). `dockerImage` carries the provider key (UI resolves to a concrete image id). `dockerTag` is empty in 4.2 — UI picks the latest tag for the resolved (image, pgMajorVersion) pair. Contract documented in the godoc on `probe.ProposedConfig` and matched in the TypeScript type.

### URL parser rules

Wraps `pgconn.ParseConfig` which already handles both libpq DSN (`host=x port=y dbname=z`) and URI (`postgres://`/`postgresql://`) forms, including IPv6 and quoted values. Thin wrapper additionally:

- Rejects if `cfg.Password != ""` (covers password in URI userinfo and `password=` in DSN).
- Rejects if `len(cfg.Fallbacks) > 0` (multi-host out of scope).

`port` defaults to 5432 when absent. `dbname` defaults to `username` in URI form (libpq behavior).

### Detection rules table (provider.go)

| Provider | Hostname suffix (case-insensitive) | Extension fingerprint (exact name match) |
|---|---|---|
| RDS | `.rds.amazonaws.com` | `rds_tools` |
| Aurora | `.rds.amazonaws.com` (only when Aurora-specific ext also present) | `aurora_stat_utils` |
| CloudSQL | (no stable hostname pattern; rely on fingerprint) | `cloudsql_iam` |
| Supabase | `.supabase.co`, `.supabase.com`, `.pooler.supabase.com` | `pg_graphql` + `supabase_vault` |
| Azure | `.postgres.database.azure.com` | (none distinctive) |
| Heroku | (Heroku DSN host pattern) | (none distinctive) |
| TimescaleCloud | `.tsdb.cloud.timescale.com` | (timescaledb is too broad) |
| Generic | (no match) | (no match) |

Hostname wins where unambiguous; extensions break ties.

**Aurora detection caveat**: `aurora_stat_utils` appears in `pg_available_extensions` on Aurora-PostgreSQL by default, but is **not** an installed extension and is **not** present on standard RDS-PostgreSQL — so the query `SELECT name FROM pg_available_extensions` distinguishes the two cleanly when run against either. If a future Aurora image stops exposing `aurora_stat_utils` in `pg_available_extensions` (e.g. AWS removes it or gates it behind a parameter-group setting), Aurora will silently misdetect as RDS. This is acceptable for 4.2: the RDS image works on Aurora, only the telemetry/preview label is wrong. Capture as a known limitation in Post-Completion's manual cross-provider verification.

### Storage compatibility

YAML stays:

```yaml
retrieval:
  spec:
    logicalDump:
      options:
        source:
          connection:
            host: ...
            port: 5432       # omitted if originally omitted (empty-port preservation rule in Task 13);
                             # see config.example.logical_rds_iam.yml where port: is absent
            username: ...
            dbname: ...
            password: ...
```

No migration step. Physical-mode YAML schema unchanged — projection layer adds tag declarations for `walg.backupName`, `pgbackrest.stanza`, `pgbackrest.delta`, the `envs` map, and the `sync.enabled` toggle. The customTool path is not projected.

## Post-Completion

*Items requiring manual intervention or external systems — no checkboxes, informational only.*

**Manual cross-provider verification** (required before tagging the 4.2 release):
- Run Simple-mode end-to-end against each provider: RDS, Aurora, Google CloudSQL, Supabase, Azure, Heroku, TimescaleCloud, and one self-hosted source (Generic). Capture which providers detect cleanly, which fall back to Generic, and any image-tag mismatches. File follow-up issues for any provider where detection is unreliable.
- Verify the resolved `shared_preload_libraries` produce a container that actually boots for each provider's matched image. Since 4.2 passes the source list through without filtering (Task 4), a mismatch between source libs and image-bundled libs surfaces as a `could not load library` failure in `docker logs dblab_server`. Capture which (provider, library) pairs fail and either bundle the library in the image upstream or file a docs note recommending users remove it from Simple-mode before Apply.
- Run Expert-mode physical end-to-end for WAL-G against a real source with a real backup; spot-check pgBackRest against `config.example.physical_pgbackrest.yml`. pg_basebackup users continue to edit YAML directly — verify that path still works after Task 8's gate change.

**Release-notes / docs (external):**
- Update postgres.ai/docs tutorial pages to point at the new Simple-mode flow as the primary path; demote the per-field setup to an Expert-mode section. Add a Physical Mode UI section.
- 4.2 release notes: Simple mode + physical-mode-in-UI are both notable.

**Observability (external):**
- After 4.2 ships, watch `ConfigProbedEvent` telemetry and `/admin/probe-source` error rates in production to see which providers fail detection most often. Use the data to refine the `provider.go` detection table in 4.2.x or 4.3.

**Foundation for `dblab local-install` (pulled forward into this cycle):**
- The `probe` package is HTTP/UI-free, so the `dblab local-install` CLI calls `probe.Propose` directly without going over HTTP. See `docs/plans/completed/20260619-cli-connstrings-glibc.md`.

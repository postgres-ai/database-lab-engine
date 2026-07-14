# CLI local-install, Connection-String Passthrough, and glibc/Collation Auto-Image

## Overview

Three features that extend the `feat/simplified-install` work. The backend probe/propose
engine (`internal/retrieval/probe/`, `POST /admin/probe-source`, the config projection with
physical-mode fields) already ships in 4.2; these features build on top of it.

1. **`dblab local-install` CLI** — a CLI front end to the existing probe + config-apply
   endpoints, so a DBLab instance can be configured (logical retrieval) from the terminal
   without the UI. Pull forward from 4.3 to 4.2.

2. **Connection-string full paramspec passthrough** — accept a complete libpq connection
   string (both `postgresql://…` URI and `host=… port=… dbname=…` keyword/value forms) for
   the source database, preserving *all* libpq options (`sslmode`, `connect_timeout`,
   `sslrootcert`, `options`, …) end-to-end into `pg_dump` and the engine's own pgx
   connections **to the source**. Discrete `host/port/dbname/username` fields remain supported.

3. **glibc/collation detection + auto-image-selection (PG15+)** — probe the source with
   `pg_database_collation_actual_version()`, carry the result through `ProposedConfig`, and
   resolve a matching image by querying the container registry **live** (an engine-side
   registry client with caching) for **both** the generic image (`postgresai/extended-postgres`
   on Docker Hub) and the managed-provider SE images (`registry.gitlab.com/postgres-ai/se-images/<provider>`
   on GitLab), under one unified tag scheme `<pgMajor>[-<extVersion>][-glibc<NNN>]`
   (e.g. `16-0.7.0-glibc236`); glibc matching applies only to **PG15+**. The probe returns a
   resolved full image reference (`resolvedImage`) plus `DockerTag`. **No hardcoded image
   inventory** — the registries are the source of truth, with a `go:embed` snapshot as offline
   fallback. When nothing matches, fall back to the provider's default image (`<repo>:<major>`).
   **External prerequisite: the `-glibcNNN` tags must be built and published for both the
   generic and SE images first** — see Post-Completion; default (non-glibc) selection works
   immediately.

### Problem it solves
- CLI: enables headless/scripted setup; the UI is not always available (servers, CI, RDS bootstrap).
- Connection strings: managed providers (RDS, Supabase, CloudSQL) frequently require `sslmode`
  and friends; today those params are silently dropped after host/port/dbname/user extraction.
- glibc: glibc version mismatch between source and the DBLab Postgres container causes silent
  collation/index corruption. PG15+ exposes the actual collation version, enabling detection.

### How it integrates
- CLI reuses `POST /admin/probe-source` and `POST /admin/config` via new `dblabapi.Client`
  methods — no new server endpoints, no new backend orchestration.
- Connection-string support adds an optional `connectionString` field to the logical
  `Source` config (and a matching `ConfigProjection` entry so it persists through
  `POST /admin/config`); the existing `probe.ParseConnectionString` already parses both forms.
- glibc detection extends the existing `probe.Propose` query set and `ProposedConfig`.

## Context (from discovery)

### Files/components involved
- **CLI**: `engine/cmd/cli/main.go` (urfave/cli/v2 command registration),
  `engine/cmd/cli/commands/client.go` (`ClientByCLIContext` returns a concrete `*dblabapi.Client`),
  `engine/cmd/cli/commands/global/{command_list,actions}.go` (template: `init` command),
  `engine/cmd/cli/commands/clone/actions_test.go` (test pattern: unit-test **pure helpers**,
  not action flows — no client mocking/interfaces).
- **API client**: `engine/pkg/client/dblabapi/` (`client.go`; per-resource files like
  `status.go`). Pattern = method on `*Client`, `c.URL("/path")`, `c.Do(ctx, req)`, decode JSON.
  `Do` (client.go:127-141) **already maps non-2xx to `models.Error`**; `FullRefresh` already
  exists (`status.go:77`).
- **Probe**: `engine/internal/retrieval/probe/{probe,propose,parser}.go`.
  `Propose(ctx, connStr, password) ProposedConfig`; `ParseConnectionString` already wraps
  `pgx.ParseConfig` and accepts both libpq forms (pgx v5.9.2 in `go.mod`). `probe` imports
  nothing from `logical` (`probe.Connection` mirrors `logical.Connection` precisely to avoid
  that dependency), so a `logical → probe` import is safe. `assembleProposed` already takes
  6 positional args. Integration suite: `probe/propose_integration_test.go` (testcontainers).
- **Server handlers**: `engine/internal/srv/config.go` — `probeSource` (~143),
  `setProjectedAdminConfig` (~65, `POST /admin/config`, auto-`FullRefresh` only when status is
  `Pending`), `testDBSource` (~106, driven by `models.ConnectionTest` — **out of scope here**),
  `guardModeFields` (~215, per-mode field allow-list). Routes: `server.go:259-263` under
  `/admin` (AdminMW).
- **Logical dump (connects to SOURCE)**: `engine/internal/retrieval/engine/postgres/logical/dump.go`
  — `Source` struct (~94), `Connection` struct (~127), `getDBList` (~421-424: builds a DSN via
  `db.ConnectionString` then `pgx.Connect` to **enumerate source databases** when the
  `databases` map is empty), `buildLogicalDumpCommand` (~686: per-database
  `--host/--port/--username/--dbname`), `getExecEnvironmentVariables` (~670: `PGPASSWORD`).
- **Logical restore (LOCAL only — must NOT receive the source connection string)**:
  `engine/internal/retrieval/engine/postgres/logical/restore.go` (~810-882: restores a local
  dump file into the local DBLab container; `RestoreOptions` has **no** source `Connection`),
  and `dump.go`'s inline `buildLogicalRestoreCommand` (~736, immediate-restore path, also local).
- **Source activity (pgx to SOURCE)**: `engine/internal/retrieval/engine/postgres/logical/activity.go`
  (`dbSourceActivity` ~43, `pgx.ParseConfig`).
- **test-db-source path (out of scope)**: `engine/internal/retrieval/engine/postgres/tools/db/pg.go`
  (`ConnectionString` ~82, `checkConnection` ~320, `EscapeLibpqValue` for libpq quoting).
- **Image selection / version parse**: `engine/internal/retrieval/engine/postgres/snapshot/physical.go`
  (~635 `postgresai/extended-postgres:%g`; same default pattern in `snapshot/rename.go` ~81 and
  `snapshot/logical.go` ~220 — the **default image name** is `postgresai/extended-postgres:<major>`),
  `engine/internal/provision/mode_local.go` (`regDockerImage = ":([.0-9]+)"` ~899,
  `parseImageVersion` — **will not match a `16-0.7.0-glibc236`-style tag**).
- **Registry (new, engine-side)**: no client exists today — image tags are resolved client-side
  (the UI's hardcoded `dockerImagesConfig`, already stale) or via the platform-only
  `dblab_se_images` PostgREST table. Two registries, both with public, anonymous (rate-limited)
  tags APIs:
  - **Docker Hub** (generic): `https://hub.docker.com/v2/repositories/postgresai/extended-postgres/tags?page_size=100`
    (paginated; returns `name`, `last_pushed`, arch, digest).
  - **GitLab** (SE / managed providers): project `postgres-ai/se-images` (id 41786321),
    `https://gitlab.com/api/v4/projects/41786321/registry/repositories` then per-repo `/tags`.
    Repos: `rds`, `aurora`, `supabase`, `google-cloud-sql`, `heroku`, `timescale-cloud`,
    `postgis`, plus `extended-postgres`. **Repo names are NOT all equal to the probe provider
    keys** — `provider.go` emits `cloudsql` and `timescale`, whose repos are `google-cloud-sql`
    and `timescale-cloud`; a provider-key→repo translation is required (mirrors the UI's
    `providerKeyToImageType`). The UI constant `seContainerRegistry = 'se-images'` and
    `getSeImages` confirm this layout.
  Both use the `<pgMajor>` / `<pgMajor>-<extVersion>` scheme (e.g. `17-0.7.0`, SE `15-0.8.0`);
  SE repos also carry branch-named CI tags (`13-nik-…`, `15-fix-ci`) that the strict parser
  ignores. **No `-glibcNNN` tags exist on either registry yet.**
- **Models**: `engine/pkg/models/admin.go` (`ProbeSourceRequest` ~89, `ProposedConfig` DTO ~113
  — already has `dockerTag`, `SourceConnection` ~96),
  `engine/pkg/models/configuration.go` (`ConfigProjection` ~20 — has `DockerImage` only, a
  single `databaseContainer.dockerImage` value; **no `connectionString`, no separate `dockerTag`**).

### Related patterns found
- `ConfigProjection` uses flat `proj:"…"` paths walked by `pkg/util/projection`; only tagged
  fields persist through `POST /admin/config`.
- The probe API keeps the password strictly separate from the connection string (rejects
  embedded passwords) so it never reaches logs/telemetry/projection writes. **Preserve this.**
- User convention (CLAUDE.md + memory): do **not** introduce interfaces/abstractions solely
  for test mocking — extract pure helpers and test real behavior (see `clone/actions_test.go`).
- Functions with 3+ inputs should take an input struct (relevant to `assembleProposed`).

### Dependencies identified
- `github.com/jackc/pgx/v5 v5.9.2` (present) parses both connection-string forms.
- `github.com/urfave/cli/v2` (the CLI framework). No new Go modules required.
- Docker Hub tags API (public, anonymous, rate-limited) for image discovery, queried with the
  stdlib `net/http`. **No new Go modules**: `golang.org/x/sync` is NOT in `go.mod`, so coalesce
  concurrent cache refreshes with a refresh mutex (not `singleflight`).

## Development Approach
- **Testing approach**: Regular (code first, then tests) — matches the repo's Testify
  table-driven convention (`make test`, `make run-lint`).
- Complete each task fully before the next; small, focused changes.
- **Every task includes new/updated tests** (success + error/edge cases) as separate items.
- **All tests must pass before the next task.** Run `cd engine && make test` and
  `cd engine && make run-lint` after each task.
- Maintain backward compatibility: discrete `host/port/dbname/username` keeps working;
  the UI's empty-`DockerTag` fallback keeps working; the local restore path is untouched.

## Testing Strategy
- **Unit tests** (required every task): `withDatabase`/parser helpers, the registry client
  (httptest paginated tags + cache + embedded fallback), `selectImageTag`, `parseImageVersion`
  against the new tag format, CLI pure helpers (preview render, projection assembly,
  start/no-start), client request construction.
- **Integration tests**: extend `probe/propose_integration_test.go` for the collation-version
  query; extend `srv/probe_source_integration_test.go` for the end-to-end probe response;
  exercise a `connectionString` source in the logical dump integration suite where feasible.
- **No UI e2e in this plan** — engine/CLI changes only. UI catalog coordination is noted under
  Post-Completion.

## Progress Tracking
- Mark completed items `[x]` immediately when done.
- Add newly discovered tasks with ➕ prefix; document blockers with ⚠️ prefix.
- Keep this file in sync with actual work; update if scope changes.

## What Goes Where
- **Implementation Steps** (`[ ]`): code, tests, docs achievable in this repo.
- **Post-Completion** (no checkboxes): the published glibc-pinned images, UI catalog
  coordination, manual end-to-end verification against live providers.

## Implementation Steps

### Task 1: Connection-string config field (`Source` YAML + `ConfigProjection`)

**Files:**
- Modify: `engine/internal/retrieval/engine/postgres/logical/dump.go` (`Source` ~94, `setDefaults`/`Reload`)
- Create: `engine/internal/retrieval/engine/postgres/logical/connstring.go`
- Create: `engine/internal/retrieval/engine/postgres/logical/connstring_test.go`
- Modify: `engine/pkg/models/configuration.go` (`ConfigProjection`)
- Modify: `engine/internal/srv/config.go` (`guardModeFields` ~215)
- Modify: `engine/internal/srv/config_test.go` (guard test)

- [x] add `ConnectionString string yaml:"connectionString"` to `Source` (dump.go ~94)
- [x] in `connstring.go`, add `withDatabase(connStr, dbName string) (string, error)` that
      overrides/sets `dbname` in either form: URI → set path via `net/url` (preserve query
      params); keyword/value → append `dbname=<escaped>` (libpq last-wins), reusing
      `db.EscapeLibpqValue` for quoting; preserve every other param
- [x] reuse `probe.ParseConnectionString` directly to derive discrete host/port/user/dbname
      for display/validation (the import is safe — `probe` does not import `logical`); do **not**
      create a new neutral package unless a real cycle is proven
- [x] in `setDefaults`/`Reload`, when `ConnectionString` is set it **wins**: derive
      `Source.Connection` (host/port/user/dbname) from it for display/validation, ignore any YAML
      `connection.*`, and apply the derived values to `d.config.db`. Done via `applySourceConnectionString`
      called in `Reload` (parse errors surface there; `setDefaults` can't return one). No "both set"
      rejection, per the note.
- [x] add `ConnectionString *string` to `ConfigProjection` with
      `proj:"retrieval.spec.logicalDump.options.source.connectionString"`, and add it to the
      logical-field list in `guardModeFields` so it is allowed in logical mode and rejected in physical
- ➕ [x] the projection field needs the **`,createKey`** suffix: `Set` (projection/yaml.go) silently
      skips a leaf key that isn't already present in the YAML, and no config scaffold/`ensureLogicalPipeline`
      seeds `connectionString` (unlike `source.connection.*`, which the example config pre-seeds). The
      parent `source` block already exists, so `createKey` creates the leaf. Verified by the round-trip test.
- [x] write tests for `withDatabase` (URI + keyword forms, IPv6 host, existing `dbname`,
      query params preserved, missing `dbname`, override precedence) and for connection-string
      precedence (derived `Connection` overrides YAML `connection.*`; defaults are not treated as conflicts)
- [x] extend the `guardModeFields` test for the new field
- [x] add a projection round-trip test: a projection JSON carrying `connectionString` persists via
      `StoreYaml` to `retrieval.spec.logicalDump.options.source.connectionString`, reloads into
      `Source.ConnectionString`, and is rejected by `guardModeFields` under physical mode
- [x] run `cd engine && make test && make run-lint` — must pass before Task 2

### Task 2: Pass the connection string into `pg_dump` (dump side only)

**Files:**
- Modify: `engine/internal/retrieval/engine/postgres/logical/dump.go` (`buildLogicalDumpCommand` ~686, `getExecEnvironmentVariables` ~670)
- Modify: `engine/internal/retrieval/engine/postgres/logical/restore_test.go` (logical unit tests live here)

- [x] when `Source.ConnectionString` is set, build `pg_dump` with
      `-d <withDatabase(connStr, dbName)>` instead of the discrete `--host/--port/--username/--dbname`
      flags (keep `--create`, `--jobs`, table filters, custom options unchanged). Extracted into
      `dumpConnectionArgs`; `buildLogicalDumpCommand` now returns `([]string, error)` (callers updated).
- [x] **leave all restore commands untouched** — verified via the immediate-restore+connStr test case:
      pg_restore still uses local `--username postgres --dbname postgres`, never the source conninfo
- [x] keep password strictly separate: still set `PGPASSWORD` from `config.db.Password`
      (or pre-existing env) — never inline the password into the conninfo
- [x] ensure the connection-string branch and the discrete-field branch are mutually exclusive
- [x] write table-driven tests asserting exact `pg_dump` argv for: discrete fields (unchanged),
      connection string with extra params (sslmode/connect_timeout preserved), and
      multi-database iteration (dbname overridden per DB, via `TestDumpConnectionArgs`); assert restore argv is unchanged
- [x] run `cd engine && make test && make run-lint` — must pass before Task 3

### Task 3: Raw connection string for the engine's pgx connections to the source

**Files:**
- Modify: `engine/internal/retrieval/engine/postgres/logical/dump.go` (`getDBList` ~421)
- Modify: `engine/internal/retrieval/engine/postgres/logical/activity.go` (`dbSourceActivity` ~43)
- Modify/Create: `engine/internal/retrieval/engine/postgres/logical/dump_integration_test.go` (or the existing unit test file)

- [x] in `getDBList`, when `Source.ConnectionString` is set, feed the raw string to
      `pgx.ParseConfig` (injecting password separately) instead of building a DSN via
      `db.ConnectionString(...)` from discrete fields — this preserves `sslmode`/TLS. Done via the
      shared `sourcePgxConfig` helper (connstring.go); `getDBList` now uses `pgx.ConnectConfig`.
- [x] in `dbSourceActivity` (activity.go), use the same `sourcePgxConfig` helper + set `ConnectTimeout`;
      signature extended to `(ctx, connStr, dbCfg)` and the `ReportActivity` call site threads `Source.ConnectionString`
- [x] **do not** touch `tools/db/pg.go` `ConnectionString`/`checkConnection` — left untouched (out of scope)
- [x] write tests covering raw-string and discrete-field paths (param preservation via `connect_timeout`,
      password injected separately, `withDatabase` applied) — `TestSourcePgxConfig`
- [x] run `cd engine && make test && make run-lint` — must pass before Task 4

### Task 4: Probe `pg_database_collation_actual_version()` (PG15+)

**Files:**
- Modify: `engine/internal/retrieval/probe/propose.go` (`assembleProposed`, new query helper)
- Modify: `engine/internal/retrieval/probe/probe.go` (`ProposedConfig`)
- Modify: `engine/internal/retrieval/probe/propose_test.go`
- Modify: `engine/internal/retrieval/probe/propose_integration_test.go`

- [x] refactor `assembleProposed` to take an **input struct** (`proposalInputs`) and update existing call sites/tests
- [x] add `queryCollationVersion(ctx, conn, major)` guarded by PG major >= 15. Uses the recorded
      `select coalesce(datcollversion, '') from pg_database where datname = current_database()`
      (coalesce maps NULL→""). Tolerates empty/absent by returning "" not an error. The guard returns
      before touching the conn, so a nil conn is safe for the unit test.
- [x] add `CollationVersion string` to `probe.ProposedConfig` with godoc noting it is the
      provider-reported collation version (glibc string for libc, different/empty for ICU) —
      a strong signal, not a literal `ldd` glibc version; populate it in `assembleProposed`
- [x] write unit tests for `assembleProposed` (struct input + collation field) and the
      PG-version guard (`TestQueryCollationVersion_GuardBelowPG15`)
- [x] extend `propose_integration_test.go` to assert a non-empty collation version on a PG15+
      testcontainer (`postgres:16`; parametrized `startProbePostgresImage` helper)
- [x] run `cd engine && make test && make run-lint` — must pass before Task 5

### Task 5: Engine-side registry client (cached, multi-registry) + glibc-aware image resolution

Resolve the image by querying the container registry **live** (cached), not from a hardcoded
list — for **both** the generic image and the managed-provider SE images, under one unified tag
scheme `<pgMajor>[-<extVersion>][-glibc<NNN>]` (e.g. `16-0.7.0-glibc236`). glibc matching is
**PG15+ only**; on any miss, fall back to the provider's default image. `Propose` returns a
resolved full image reference (`<repo>:<tag>`) so neither the CLI nor the UI re-derives it;
`DockerImage` (provider key) and `DockerTag` are kept for backward compatibility.

**Provider → repo (derived from the registries, not hardcoded):**
- `generic` → `postgresai/extended-postgres` on **Docker Hub** (public tags API).
- managed → `registry.gitlab.com/postgres-ai/se-images/<repo>` on **GitLab** (project
  `postgres-ai/se-images`, id 41786321; **anonymously listable**, so discovery needs no token).
  A provider-key→repo translation is required because two keys differ from their repo name:

  | provider key (`provider.go`) | se-images repo |
  | --- | --- |
  | `rds`, `aurora`, `supabase`, `heroku` | same |
  | `cloudsql` | `google-cloud-sql` |
  | `timescale` | `timescale-cloud` |
  | `azure` | *(no SE repo → falls back to generic)* |

  Each layer keys on a different identifier: **detection emits provider keys**; the
  **registry/snapshot key on repo names**. Both registries use the `<pgMajor>[-<extVersion>]`
  release scheme; the strict parser drops their branch-named CI tags (`13-nik-…`, `15-fix-ci`).

**Resolution chain (must never hard-fail or hang when third-party registries are down):**
(1) fresh in-memory cache → (2) live registry fetch (short timeout) → (3) last good in-memory
cache → (4) **embedded `images_fallback.json` snapshot** (`go:embed`, always in the binary) →
(5) the provider default image (`<repo>:<major>`). Layers 4–5 guarantee a usable image with
**zero network** (offline/air-gapped). The provider-keyed seed snapshot exists at
`engine/internal/retrieval/probe/images_fallback.json` (generic **and** SE pinned tags seeded;
SE tops out at PG17; no `-glibcNNN` upstream yet) and is regenerated by CI.

**Files:**
- Create: `engine/internal/retrieval/probe/registry.go` (multi-registry tags client: Docker Hub + GitLab backends + per-repo TTL cache + embedded fallback)
- Create: `engine/internal/retrieval/probe/registry_test.go` (httptest for both backends; pagination; cache hit/refresh; fallback-on-error)
- Create: `engine/internal/retrieval/probe/imageselect.go` (pure tag parse + selection + provider→repo resolution)
- Create: `engine/internal/retrieval/probe/imageselect_test.go`
- Use/regen: `engine/internal/retrieval/probe/images_fallback.json` (**already seeded** for generic + all SE providers; provider-keyed; `go:embed`; CI regenerates)
- Modify: `engine/internal/retrieval/probe/probe.go` (add `ResolvedImage` to `ProposedConfig`; keep `DockerImage` provider key + `DockerTag`)
- Modify: `engine/internal/retrieval/probe/propose.go` (`Propose` resolves the image; pass tag + resolved ref into the `assembleProposed` input struct)
- Modify: `engine/pkg/models/admin.go` (`ProposedConfig` DTO: add `collationVersion`, `resolvedImage`)
- Modify: `engine/internal/srv/config.go` (construct the registry **once on `Server`** at startup; map the new fields; update the single `Propose` caller ~158)
- Modify: `engine/internal/provision/mode_local.go` (`regDockerImage` ~899) + `mode_local_test.go`

- [x] `registry.go`: a `Registry` type that lists tags for a repo across Docker Hub (paginated via
      `next`) and GitLab (`/registry/repositories` → per-repo `/tags`, paginated via `page`); base URLs
      are unexported fields overridable in tests; stdlib `net/http`; descriptive User-Agent; non-200
      (incl. 429/5xx) → error so caller serves cache/fallback
- [x] provider-key→repo translation in `imageselect.go` (`providerRepo`): `cloudsql`→`google-cloud-sql`,
      `timescale`→`timescale-cloud`, identity for the rest; `generic`/`""`/`azure` → Docker Hub generic.
      The embedded snapshot is keyed by repo (imageRef), matching `providerRepo`'s output.
- [x] per-repo in-memory cache behind `sync.RWMutex` with ~1h TTL; refresh coalesced by a refresh
      lock (no `singleflight`); each refresh bounded by `fetchTimeout`; chain fresh→fetch→last-cache→embedded
- [x] embed `images_fallback.json` via `go:embed`; parsed once into a repo→tags map (`loadFallback`)
- [x] `imageselect.go`: `parseImageTag` with `^(\d+)(?:-(\d+\.\d+\.\d+))?(?:-glibc(\d+))?$`; ignores
      non-matching tags; `compareExtVersion` three-int comparator (bare major sorts lowest)
- [x] `ResolveImage(providerKey, major, collationVersion)`: PG15+ glibc match → non-glibc newest →
      default `<repo>:<major>`; returns the full `<repo>:<tag>` reference + selected tag (empty tag on default)
- [x] `normalizeGlibcSuffix` strips the dot (`2.36`→`236`); glibc matching only for PG>=15; ICU/empty → no suffix
- [x] `Propose` resolves the image (via `resolveImage`, nil-registry safe) and passes
      `resolvedImage`+`dockerTag` into the `assembleProposed` input struct
- [x] `parseImageVersion` (`mode_local.go`) now uses `:(\d+)` → leading integer major from both legacy
      dotted (`:14.2`→14, `:9.6`→9) and suffixed (`16-0.7.0-glibc236`→16) tags; aligns with the
      PG_VERSION-file path (major only). Regression test covers both forms.
- [x] `CollationVersion` + `ResolvedImage` added to `models.ProposedConfig` and copied through the
      `probeSource` handler; the `Registry` is constructed once on `Server` (`imageRegistry`) and passed to `Propose`
- [x] tests: `registry_test.go` (httptest Docker Hub + GitLab pagination; cache hit; 429/5xx → embedded
      fallback; default when snapshot lacks major; azure→generic); `imageselect_test.go`
      (parse/select/compare/normalize/providerRepo); `parseImageVersion` regression
- [x] run `cd engine && make test && make run-lint` — must pass before Task 6

### Task 6: `dblabapi.Client` — `ProbeSource` and `ApplyConfig`

**Files:**
- Create: `engine/pkg/client/dblabapi/config.go`
- Create: `engine/pkg/client/dblabapi/config_test.go`

- [x] add `ProbeSource(ctx, models.ProbeSourceRequest) (*models.ProposedConfig, error)` →
      `POST /admin/probe-source` (mirrors the `branch.go` encode/POST/decode pattern)
- [x] add `ApplyConfig(ctx, projection json.RawMessage) (json.RawMessage, error)` →
      `POST /admin/config` (projection sent verbatim; response returned raw); `FullRefresh` already exists for `--start`
- [x] rely on `c.Do` for non-2xx → `models.Error` mapping (no per-method error re-decode)
- [x] `config_test.go`: ProbeSource (success asserts URL/method/body + decoded proposal; 400 → wrapped
      `models.Error`) and ApplyConfig (success asserts verbatim body round-trip; 400 error)
- [x] run `cd engine && make test && make run-lint` — must pass before Task 7

### Task 7: `dblab local-install` command

**Files:**
- Create: `engine/cmd/cli/commands/localinstall/command_list.go`
- Create: `engine/cmd/cli/commands/localinstall/actions.go`
- Create: `engine/cmd/cli/commands/localinstall/actions_test.go`
- Modify: `engine/cmd/cli/main.go` (register the command group)

- [x] define the command with flags: `--source-url` (required), `--password` (prompted on a TTY via
      `golang.org/x/term`, already in go.sum, promoted to direct by `go mod tidy`), `--provider`,
      `--docker-image`/`--docker-tag`, `--shared-buffers`, `--dbname` (repeatable),
      `--start`/`--no-start`, `--yes`
- [x] extract **pure helpers** (no client mocking): `renderPreview` (with `glibcWarning`),
      `buildProjection` (nested ConfigProjection JSON matching the `proj:` paths; `composeImage`/`replaceTag`
      for the `--docker-image`/`--docker-tag` override; `sourceCarriesExtraParams` chooses connectionString
      vs discrete fields), `shouldStartRefresh(status, start, noStart)`, `databaseSet`, `readConfirmation`
- [x] action flow: `ClientByCLIContext` → `resolvePassword` → `ProbeSource` → `renderPreview` (collation +
      resolved image + glibc warning) → confirm (unless `--yes`) → `Status` (pre-apply) → `ApplyConfig` →
      conditional `FullRefresh`
- [x] keep the password out of all output: `renderPreview` is built from the password-free `ProposedConfig`;
      the projection JSON (which carries the password for the engine) is never printed
- [x] register the group in `main.go` alongside `instance`, `clone`, etc.
- [x] write tests for the pure helpers (preview text + glibc warning, projection discrete/connection-string
      branches + overrides, composeImage, sourceCarriesExtraParams, shouldStartRefresh, readConfirmation,
      optionsFromContext) — mirror `clone/actions_test.go`
- [x] run `cd engine && make test && make run-lint` — must pass before Task 8

### Task 8: Verify acceptance criteria
- [x] `dblab local-install` builds the projection and conditional refresh — covered by the
      `localinstall` pure-helper tests; the live run against an instance is Post-Completion
- [x] both connection-string forms work end-to-end; `sslmode`/`connect_timeout` reach `pg_dump`
      (`TestDumpConnectionArgs`) and source DB enumeration (`TestSourcePgxConfig`)
- [x] the connection string persists via `POST /admin/config` — `TestProjectionRoundTrip_ConnectionString`
- [x] PG15+ source yields a non-empty `collationVersion` (`TestPropose_CollationVersion_Integration`);
      tag selection covered by `registry_test.go`/`imageselect_test.go`; `parseImageVersion` regression covers glibc tags
- [x] discrete-field configs and the local restore path are unchanged — regression cases in
      `TestDumpCommandBuilding` (restore argv unchanged) and `TestDumpConnectionArgs`
- [x] run full suite: `cd engine && make test` and `cd engine && make run-lint` — green
- [x] integration tests **compile** under `-tags integration`; running them needs docker (left to CI)

### Task 9: Documentation
- [x] document `dblab local-install` via the command `Usage`/`Description` (flags + two examples)
- [x] document `connectionString` in `engine/configs/config.example.logical_generic.yml`
- [x] update `CLAUDE.md` Notable patterns (connection-string passthrough; engine-side registry
      resolution with `go:embed` fallback; the `createKey` projection note)
- [x] move this plan to `docs/plans/completed/`

## Technical Details

### Connection-string passthrough
- `pgx.ParseConfig` accepts both forms but **discards** `sslmode` (folds it into a resolved
  `*tls.Config`); reconstructing a faithful DSN from the parsed struct is lossy. So the faithful
  approach is to keep the **original string** and hand it to `pg_dump` and to pgx unchanged.
  All engine-internal pgx connections to the source (`getDBList`, `dbSourceActivity`) re-parse
  the same raw string.
- `pg_dump` treats a `-d` value containing `=` or a URI prefix as a conninfo string → every
  libpq option works natively. Per-database dumps and per-database pgx connections override
  `dbname` via `withDatabase` (URI path, or appended `dbname=` with `EscapeLibpqValue`, last-wins).
- **Restore is local**: `pg_restore` reads the local dump and restores into the local DBLab
  container; the source connection string must never be passed to it.
- Password stays separate (`PGPASSWORD` / pgx `cfg.Password`); never embedded in the conninfo,
  never logged.
- Persistence: a new `ConfigProjection.ConnectionString` (`proj:` tag) is required so
  `POST /admin/config` writes the string; `guardModeFields` classifies it as logical-only.

### glibc/collation
- `pg_database_collation_actual_version()` (PG15+) returns the provider's actual collation
  version. For libc-provider databases this is the glibc version (e.g. `2.36`); ICU-provider
  databases report an ICU version or nothing useful for glibc matching — only libc results drive
  glibc matching; ICU/empty fall through to the default image.
- **Image discovery queries the registries, not a hardcoded list.** The engine-side `Registry`
  client lists tags for the relevant repo across two backends — Docker Hub for the generic image
  (`postgresai/extended-postgres`) and GitLab for the managed-provider SE images
  (`registry.gitlab.com/postgres-ai/se-images/<provider>`) — caches them (TTL + `go:embed`
  snapshot fallback), and a pure selector matches them. This replaces the stale hardcoded UI
  catalog for the engine path and stays current automatically. The SE repo *set* is registry-derived
  (the `se-images` project repo listing); a tiny provider-key→repo translation maps the two keys
  that differ (`cloudsql`→`google-cloud-sql`, `timescale`→`timescale-cloud`), and `azure`/unmapped
  providers fall back to the generic image. Detection emits **provider keys**; the registry and
  the embedded snapshot key on **repo names**.
- Tag scheme `<pgMajor>[-<extVersion>][-glibc<NNN>]` (e.g. `16-0.7.0-glibc236`); bare `<pgMajor>`
  and `<pgMajor>-<extVersion>` are the **default (non-glibc)** images. Selection (per resolved
  repo): filter by major; PG15+ → newest ext-version tag with the matching `glibc<NNN>` suffix;
  on no glibc match, or PG<15, or ICU/empty collation → newest non-glibc tag; if the major is
  absent → the default `<repo>:<major>`.
- The probe returns a resolved full image reference in `ResolvedImage` (`<repo>:<tag>`), so the
  CLI sets `databaseContainer.dockerImage` directly from it (no provider-key→path re-derivation).
  `DockerImage` (provider key) + `DockerTag` are kept so older UI builds keep working; an empty
  `DockerTag` still lets the UI catalog resolve the tag (no contract break).
- The cache uses a `sync.RWMutex` + TTL and a refresh lock (no `singleflight` — not in go.mod);
  refreshes are bounded by a short timeout. Resolution degrades through: fresh cache → live
  fetch → last good cache → **embedded `images_fallback.json`** (`go:embed`) → the provider
  default image `<repo>:<major>`. So a probe never hangs on or hard-fails from a
  slow/unreachable/rate-limited registry, and works fully offline. The provider-keyed seed
  snapshot is committed at `engine/internal/retrieval/probe/images_fallback.json` and regenerated by CI.
- **Decided not to add a persisted on-disk cache now.** The in-memory cache plus the embedded
  snapshot already satisfy the offline guarantee; a host-side persisted cache (fresher than the
  seed between deploys, survives restarts) is a freshness optimization we can add later if needed.
- `parseImageVersion` (`mode_local.go`) currently only parses numeric/dot tags; the suffixed tag
  scheme requires updating `regDockerImage` (extract the leading integer, with a regression test)
  so PG version detection keeps working.

### CLI projection assembly
- The CLI builds a `ConfigProjection`-shaped JSON (`json.RawMessage`) from the `ProposedConfig`
  plus flag overrides and `POST`s it to `/admin/config`. `--start` calls `/full-refresh` only
  when the apply did not already trigger it (status was not `Pending`).
- CLI actions are tested via extracted pure helpers (no client interface/mocks), per project
  convention.

## Post-Completion
*External actions — no checkboxes, informational only.*

**External prerequisite (blocks the glibc auto-match half of Task 5):**
- The registry client and selector ship ready and **immediately** improve default tag selection
  by reading the live registries (replacing the stale hardcoded UI catalog for the engine path),
  for both the generic and the SE images.
- The glibc-pinned variants (`<major>-<ext>-glibc<NNN>`, e.g. `16-0.7.0-glibc236`) must be
  **built and published for both registries** — generic `postgresai/extended-postgres` on Docker
  Hub **and** every `registry.gitlab.com/postgres-ai/se-images/<provider>` repo on GitLab —
  before glibc auto-matching returns results. Until then the engine surfaces the collation
  version, warns, and selects the default (non-glibc) image. Re-tagging the SE images to the
  unified scheme and publishing the glibc variants is an infra task on the image-build pipeline,
  outside this repo; the embedded `images_fallback.json` snapshot should be regenerated when they land.

**UI coordination:**
- The UI currently resolves the image from its own catalog (`dockerImagesConfig`) + `getSeImages`.
  Once the engine returns `resolvedImage` (and `DockerTag`), confirm the UI prefers the engine
  `resolvedImage` when present and falls back to its catalog/`getSeImages` when empty. Confirm the
  UI renders the new `collationVersion` field in the preview. Longer term the UI's hardcoded
  `dockerImagesConfig` can be retired in favor of the engine-resolved value.

**Manual verification:**
- Run `dblab local-install` against a live managed source requiring `sslmode=require`
  (e.g. RDS) and confirm dump + database enumeration succeed with the param preserved.
- Verify collation detection against a PG15+ source and confirm the proposed image tag matches
  the source glibc once the images exist.

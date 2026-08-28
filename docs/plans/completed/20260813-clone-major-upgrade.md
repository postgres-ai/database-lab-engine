# Clone Major Upgrade

## Overview
- Add an in-place PostgreSQL major upgrade action for an existing clone: `POST /clone/{id}/upgrade` + `dblab clone upgrade <id> --target-version N` + UI action.
- Solves: testing new PG major versions against production-like data in DBLab. Today every clone runs the single engine-wide `dockerImage`; there is no way to try PG 17 when the instance runs PG 16.
- Flow: pull images → gracefully stop the clone's Postgres → one-off upgrade container runs `pg_upgrade --link` on the clone's thin-clone dataset → clone restarts on the target-version image. Reset returns the clone to the original version (reset = rollback).
- **Out of scope (iteration 1)**: creating new clones from a committed upgraded snapshot. `StartSession` always uses the engine-default image (`mode_local.go:722`), so a clone from an upgraded snapshot would get the old major against new data and fail to start. Follow-up: record the major/image on the snapshot (or detect `PG_VERSION` in the snapshot data dir) and resolve the image in `StartSession`. The commit action itself is not blocked in iteration 1, but the limitation must be documented.

## Context (from discovery)
- Clone lifecycle: `POST /clone` → `cloning.Base.CreateClone` (`engine/internal/cloning/base.go:150`) → `Provisioner.StartSession` (`engine/internal/provision/mode_local.go:154`) → `fsm.CreateClone` (ZFS) → `postgres.Start` container.
- Container image comes from the engine-wide `provision.Config.DockerImage`; per-clone container config is `resources.AppConfig` (`engine/internal/provision/resources/appconfig.go`, has a `DockerImage` field already) and the container is launched via `docker run` string assembly in `engine/internal/provision/docker/docker.go` (`RunContainer`). When DLE itself runs in a container, `RunContainer` derives volumes from DLE's own mounts (`getMountVolumes`/`buildVolumesFromMountPoints`, `docker.go:129-175`) — never binds arbitrary host paths directly.
- **`postgres.Stop` does not shut Postgres down** — it is `docker container rm --force --volumes` (`postgres.go:141` → `docker.go:215`), i.e. SIGKILL leaving PGDATA in crash state. Graceful shutdown exists at `tools.StopPostgres` (`engine/internal/retrieval/engine/postgres/tools/tools.go:349`: `pg_ctl -D <dataDir> -w stop` exec'd as `postgres`); `Provisioner` has `p.dockerClient`.
- **Clone data directory ownership has two writers, so it must be read, not assumed.** Clone containers run with no `--user` (`docker.go:76-88`); `extended-postgres` inherits the official image entrypoint, which chowns PGDATA and re-execs via `gosu postgres` (uid 999) whenever it starts as root. Independently, `zfs.go:214` chowns the clone dataset on the host to `osUsername`, which `pool/manager.go:157-167` takes from `user.Current()` — the engine process's own user, root under containerized DLE. Neither is a reliable predictor on its own, so any additional container touching the dataset resolves `--user` by stat'ing the data directory (`dataDirOwner`), and refuses to proceed when it is root-owned since pg_upgrade will not run as root.
- **DBLab-managed config lives inside the data dir** and is applied by `pgconfig` `init()` (`pgconfig/configuration.go:106-149`, `:202-224`): `postgresql.conf` rewritten to include-only (guarded by the `initializedLabel` first line), `postgresql.dblab.*` files copied from `configs/standard/postgres/default/<major>/`, `pg_hba.conf` from `configs/standard/postgres/control/`. `postgres.Start` does NOT re-apply it (only `ApplyUserConfig` when `ExtraConf()` is set, `postgres.go:48-57`). A fresh initdb'd data dir has none of it and must be corrected explicitly via `pgconfig.NewCorrector(dataDir)`. Default config dirs exist for majors 10-18; **a major without a directory there cannot be an upgrade target**.
- User-supplied clone config is re-applied through `appConfig.SetExtraConf(session.ExtraConfig)` before `postgres.Start` (`mode_local.go:286` in `ResetSession`), which regenerates `postgresql.dblab.user_defined.conf`. No file needs to be copied by hand.
- `ResetClone` (`base.go:530`) is the orchestration pattern to mirror: pre-checks → status update → async goroutine calling provisioner → status OK/FATAL → `SaveClonesState()` → webhook + telemetry events.
- Clone state persists across engine restarts in `sessions.json` (`engine/internal/cloning/storage.go`); new fields on `models.Clone` serialize automatically. **`filterRunningClones` (`storage.go:77`) deletes a wrapper when its status is `StatusFatal` OR its container is not running**, and `cleanupInvalidClones` → `StopAllSessions` → `fsm.DestroyClone` then destroys the dataset of everything dropped. So a clone left `FATAL` with no container is destroyed at the next restart — an interrupted upgrade must not be parked in that state.
- Startup order in `Base.Run` (`base.go:88-104`): `RestoreClonesState()` → `restartCloneContainers` → `filterRunningClones` → `cleanupInvalidClones`. A recovery step inserted right after `RestoreClonesState()` runs before any of the destructive filtering.
- Idle-clone sweeper: `isIdleClone` (`base.go:879-916`) exempts protected / `EXPORTING` / dependent-snapshot / recently-started clones; `UPGRADING` needs the same exemption. `destroyPreChecks` (`base.go:341`) guards only protection.
- PG major version is parsed from the image tag (`DetectDBVersion`/`parseImageVersion`, `mode_local.go:913-944`). `parseImageVersion` matches `:(\d+)` and takes the **last** match, so registry hosts with ports survive. `DetectDBVersion` is instance-level only (`cmd/database-lab/main.go:223`) and reads the pool data dir — it never reflects a clone's version. `models.Database` has no version field, so today a clone's major is not exposed by the API at all.
- Probe `Registry.ResolveImage(ctx, providerKey, major, collationVersion)` (`probe/registry.go:86`) resolves by provider key, not by repo — unusable for "same repo, new major" without a reverse mapping. Not used in iteration 1.
- Status codes live in `engine/pkg/models/status.go` (`CREATING`, `RESETTING`, `EXPORTING`, `FATAL`, `WARNING`, ...). `StatusWarning` exists but is currently used only for instance status; clone status never takes it. Clone messages in the same file. Observer lives on `srv.Server` (`server.go:63`) with `GetObservingClone(cloneID)` (`observer.go:203`); `srv.Server.provisioner` is unexported but reachable from `routes.go` (same package) — the provision config itself is private, so config reads need an accessor.
- Telemetry event constants live in `engine/internal/telemetry/telemetry.go` (`CloneResetEvent`, `:27`); webhook constants in `engine/internal/webhooks/events.go`.
- API client: `engine/pkg/client/dblabapi/clone.go` (`ResetClone`/`ResetCloneAsync` pattern). `watchCloneStatus` (`clone.go:146-155`) bounds the poll loop with `c.requestTimeout` **only when the caller's ctx carries no deadline**; `requestTimeout` defaults to `defaultPollingTimeout` (60s, `client.go:52`) and is already user-settable via `Options.RequestTimeout` / the CLI `--request-timeout` flag (`cmd/cli/main.go:68`). No new timeout knob is needed — a ctx deadline is enough.
- CLI: `engine/cmd/cli/commands/clone/{actions.go,command_list.go}` (no test file yet). CI image jobs live in `engine/.gitlab-ci.yml` (`.job_template` at :180 + `engine/scripts/ci_docker_build_push.sh`), pushing to the GitLab registry; the template hardcodes `artifacts: engine/bin`. `postgresai/*` Docker Hub images are built elsewhere.
- UI CE API bindings: `ui/packages/ce/src/api/clones/`; shared clone entity `ui/packages/shared/types/api/entities/clone.ts`, status mapping `ui/packages/shared/utils/clone.ts`. `STATUS_CODE_TO_TYPE` has no `UPGRADING`/`WARNING` entry, and `checkIsCloneStable` treats only `OK`/`FATAL` as terminal — an unmapped terminal status makes the clone page poll forever. The `Status` component already supports a `'warning'` type (`components/Status/index.tsx:16`). The closed Platform app implements the same shared `Api` interface — new API fields must be optional (like `resetClone?:`).
- No existing `pg_upgrade` code anywhere in the repo.

## Development Approach
- **testing approach**: Regular (code first, then tests in the same task)
- complete each task fully before moving to the next
- make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
  - tests are not optional - they are a required part of the checklist
  - unit tests for new and modified functions
  - cover both success and error scenarios
  - keep orchestration logic in pure, unit-testable helpers; do not introduce interfaces solely for mocking
- **CRITICAL: all tests must pass before starting next task** (`cd engine && make test`)
- **CRITICAL: update this plan file when scope changes during implementation**
- run `make fmt`, `make run-lint`, `make test` after each task
- maintain backward compatibility: instances without `pgUpgradeImage` configured keep working; the endpoint (not engine startup) rejects requests when the feature is unconfigured

## Testing Strategy
- **unit tests**: required for every engine task; table-driven with Testify, compact single-line case structs. Where the repo has no pattern for a layer (handler tests, CLI tests), extract pure helpers (request validation, argument mapping, stage→safety mapping) and test those.
- **shell**: the upgrade script itself is not unit-tested in CI (explicit exception); all parameter validation and env construction happens in Go and is tested there — the script stays thin. Its one non-trivial contract (stage file + exit codes) is consumed by a pure Go function that is tested.
- **UI**: type-level correctness only; do not run `pnpm install` locally — leave UI build/tests to CI
- **e2e**: manual verification against the local test instance (see Post-Completion); an automated `engine/test/` scenario is out of scope for this MR

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered tasks with ➕ prefix
- document issues/blockers with ⚠️ prefix
- keep plan in sync with actual work done

## Decisions taken up front
These were open questions during discovery; they are settled here so no implementation task blocks on them.

- **Upgrade image ownership**: built in this repo by a dedicated job in `engine/.gitlab-ci.yml`. The `.job_template` anchor is not reused (it hardcodes `artifacts: engine/bin`); the new `.pg_upgrade_job_template` declares no artifacts. Feature and master pipelines push to the GitLab registry; release tags push `postgresai/pg-upgrade` to Docker Hub with the `DH_CI_REGISTRY_*` credentials that `build-image-latest-client` already uses — contrary to an earlier assumption in this plan, Docker Hub credentials *are* wired into this project's CI, so publication is not a manual step. Flip to "build in the docker-images repo" only if the image ends up needing the same base-image plumbing as `extended-postgres`; the Dockerfile+script stay here as the source of truth either way.
- **Old-major coverage per image**: the four majors below the target (e.g. target 18 → old bindirs for 14, 15, 16, 17). Broader coverage multiplies image size for little benefit; a jump from further back is rejected at the endpoint with an actionable message.
- **Old-side packages**: the PGDG `postgresql-<major>` server package alone. Debian ships the contrib modules inside it since PostgreSQL 10, and `--old-bindir` only needs `postgres`/`pg_ctl`/`pg_controldata`/`pg_resetwal` — `pg_dump` always comes from `--new-bindir`.
- **Statuses**: an upgrade never leaves a clone `FATAL`-with-no-container (that state is destroyed at the next engine restart, see Context). Terminal states are `OK` (upgraded), `WARNING` (upgrade did not happen or was rolled back; clone is running and usable), and `FATAL` only when recovery itself failed.

## Solution Overview
- **UX**: upgrade acts on a running clone. Request: `{"targetVersion": 17, "dockerImage": "<optional explicit image>"}`. Async operation (like reset): clone status goes `UPGRADING` → `OK` / `WARNING` (or `FATAL` only if recovery failed), with an actionable message pointing to the pg_upgrade log inside the clone dataset.
- **Mechanism**: pull both images up front (`docker.PrepareImage`) while the clone still runs → collect pre-flight data from the running clone → graceful Postgres shutdown (`pg_ctl stop`; pg_upgrade refuses a source cluster not shut down cleanly) → remove the container → `pg_upgrade --link` in the upgrade container → re-apply DBLab-managed config to the new data dir → start the container on the target image. Hard links stay within the clone's ZFS dataset — near-instant, no extra space.
- **Rollback is decided from the data directory, not from a script breadcrumb.** pg_upgrade renames `global/pg_control` out of the way as its last act, which is precisely when the old cluster stops being startable, so recovery keys on that marker plus the target-major `PG_VERSION` in `data`/`data_new`. Exit codes are a hint only. This removes the "unknown failure destroys the clone" over-classification: an intact `pg_control` means rollback is safe no matter how the run ended.
- **The engine owns every directory operation inside the clone directory.** The clone directory belongs to the engine OS user while the upgrade container runs as the data-directory owner (a different account in the standard deployment), so the container cannot create `data_new` or rename anything. The engine pre-creates the empty `data_new`, chowns it and the `upgrade/` working area to that owner, and performs the swap itself. Verified against a root-owned clone directory with a postgres-owned data directory.
- **Superseded — kept for context. Rollback was stage-dependent, not "best-effort"**. `pg_upgrade --link` makes the old cluster unsafe to start once file transfer has begun. The script therefore records a stage and exits with a stage-classified code, and the engine only ever restarts the old cluster when the failure happened *before* transfer:
  - failure at `prepare`/`initdb`/`check` → old data dir untouched → remove `data_new`, restart the old image → `WARNING`, no data loss
  - failure at `transfer`/`swap`, or an unclassifiable exit → old cluster must not be started → the engine re-provisions the clone from its origin snapshot on the original image (the `ResetSession` path) → `WARNING` stating that writes made since cloning are lost; if that re-provision fails → `FATAL`
- **A converted-but-not-yet-promoted cluster is an unfinished success, not a loss.** pg_upgrade disables the old cluster before the engine swaps directories, so that window would otherwise read as unrecoverable; `NewClusterReady` makes the engine finish the swap instead. Observed directly in the container test (`data=16 data_new=17`).
- **Interruption by an engine restart uses the same logic.** A new `recoverInterruptedUpgrades` step runs immediately after `RestoreClonesState()` and before `restartCloneContainers`/`filterRunningClones`, so recovery happens before any destructive filtering and no special-casing is needed in the filters themselves:
  - stage `prepare`/`initdb`/`check` → safe rollback, restart old image → `WARNING`
  - stage `transfer`/`swap` → auto re-provision from the origin snapshot → `WARNING`
  - stage `done` (script finished, engine died before recording success) → finish the upgrade: set the target image, start the container, delete `data_old` → `OK`
  Auto re-provisioning without asking is the right default here specifically because nobody is watching: the alternative is the clone being silently dropped and its dataset destroyed by `cleanupInvalidClones`.
- **Binaries**: per-target-major upgrade images derived FROM `postgresai/extended-postgres:<new>` (extension set for the new major matches by construction — pg_upgrade's loadable-library check needs new-major `.so`s), plus PGDG server+contrib packages for the four preceding majors (old bindirs). Tagged e.g. `pg-upgrade:17-<extver>`. Old-side `shared_preload_libraries` handled by passing `-o "-c shared_preload_libraries=..."` scoped to libraries present in the image, or emptied for the upgrade run if safe.
- **Target image resolution** (one path only): explicit `dockerImage` in the request wins; otherwise substitute only the major in the current image tag, preserving extension-bundle and glibc suffixes (`postgresai/extended-postgres:16-0.6.2-glibc236` → `17-0.6.2-glibc236`). No live registry resolution in iteration 1 — the probe `Registry` has no repo-based lookup, and silently changing the glibc build across an upgrade risks collation-dependent index corruption (pg_upgrade does not reindex).
- **Pre-flight data**: the new cluster's `initdb` must match the old cluster. Collected from the *running* clone before stopping it via `cloning.ConnectToClone`: `pg_encoding_to_char(encoding)`, `datcollate`, `datctype`, and on PG15+ `datlocprovider` + ICU locale from `pg_database` (template0), plus `SHOW data_checksums`, plus a non-default tablespace check (see Technical Details).
- **Persistence**: `models.Clone.DockerImage` records the per-clone image override and `models.Clone.DBVersion` the major actually running; both persisted via `sessions.json`; both cleared/reset on reset. Engine restarts reuse the existing Docker container (`StartCloneContainer`), so the override survives naturally.
- **Safety guards**: `UPGRADING` clones are exempt from the idle sweeper and refuse destroy/reset. The restart path needs no filter changes because recovery runs first.

## Technical Details
- Data dir layout inside the clone dataset (`<pool>/clones/<branch>/<name>/r<rev>/`): current data at `<...>/data` (`AppConfig.DataDir()`). Upgrade artifacts sit next to it in the clone dir (`AppConfig.CloneDir()`):
  - `upgrade.state.json` — written by the engine before the container starts: old major, target major, old image, target image, origin snapshot ID. Removed on success.
  - `upgrade.stage` — single word, overwritten by the script at each transition: `initdb` → `check` → `transfer` → `swap` → `done`. The engine writes `prepare` into it before starting the container. Removed on success.
  - `upgrade_logs/` — pg_upgrade output, kept after success for diagnostics.
- Upgrade steps performed by the script:
  1. `initdb` new cluster at `data_new` with matching `--encoding/--lc-collate/--lc-ctype` (+ locale provider/ICU locale on PG15+, `--data-checksums` if the old cluster has them)
  2. copy `postgresql.auto.conf` from `data` to `data_new`, **filtering out settings the new binary does not recognize** (`<new-bindir>/postgres -D data_new -C <name>` per entry; auto.conf is normally 0-5 lines). Without this, a GUC removed in the new major (e.g. `wal_keep_segments`, gone in 13) makes the new server refuse to start.
  3. `pg_upgrade --check`, then the real run with `--link --old-bindir /usr/lib/postgresql/<old>/bin --new-bindir /usr/lib/postgresql/<new>/bin`
  4. swap: `data` → `data_old`, `data_new` → `data`
  5. `data_old` is deleted by the engine after the upgraded container starts successfully (files are hard-linked, deletion reclaims only unshared files)
- **Script exit contract** (consumed by a pure Go classifier, unit-tested):
  - `0` — success, stage `done`
  - `10` — failed before any file transfer; `data` untouched and safe to start
  - `20` — failed at or after file transfer; `data` must not be started
  - anything else (docker error, OOM kill, crash before the trap) — the engine falls back to reading `upgrade.stage`; an unreadable/missing stage file is classified unsafe
  On `10` the script itself removes `data_new`; on `20` it removes nothing, leaving the directory state for diagnostics.
- After the swap and before container start, the engine re-applies DBLab-managed config to the new data dir with `pgconfig.NewCorrector(newDataDir)` (fresh-dir `init()` path — the initdb'd `postgresql.conf` lacks the `initializedLabel` line, so the full path runs), then sets `appConfig.SetExtraConf(session.ExtraConfig)` so `postgres.Start` regenerates `postgresql.dblab.user_defined.conf`, exactly as `ResetSession` does. Nothing is hand-copied from `data_old`.
- The target major must have a directory under `configs/standard/postgres/default/<major>/` (10-18 today); the endpoint rejects a target without one rather than failing after the data has been converted.
- **Tablespaces**: non-default tablespaces live outside PGDATA, therefore outside the clone dataset and outside the derived volume mounts. The pre-flight query checks `SELECT spcname FROM pg_tablespace WHERE spcname NOT IN ('pg_default','pg_global')` and the endpoint refuses the upgrade when any exist, with an explicit message. Supporting them is out of scope.
- Old/new major versions: old read from `<data>/PG_VERSION` (authoritative even after a prior upgrade), new from the request. Validation: `target > current`, `target - current <= 4` (image coverage), target has a default config dir.
- Upgrade container: one-off `docker run --rm` with parameters via env (`CLONE_DIR`, `OLD_VERSION`, `NEW_VERSION`, `ENCODING`, `LC_COLLATE_VALUE`, `LC_CTYPE_VALUE`, `LOCALE_PROVIDER`, `ICU_LOCALE`, `DATA_CHECKSUMS`, `OLD_SERVER_OPTIONS`, `DATA_SUBDIR`). `DATA_SUBDIR` carries the pool's configurable `dataSubDir` so the script never hardcodes `data`. The collation values deliberately do **not** use the names `LC_COLLATE`/`LC_CTYPE`: those are real POSIX locale variables, and exporting them would change the process locale of `initdb` and `postgres` inside the container instead of just carrying data. `OLD_SERVER_OPTIONS` is the `-o` passthrough that scopes the old server's `shared_preload_libraries` for pg_upgrade's own old-cluster start. It runs with `--user <numeric uid of the pool osUsername, resolved on the host>` (pg_upgrade refuses root, and the clone dataset is chowned to that uid) and `--workdir <clone dir>` — pg_upgrade writes `pg_upgrade_output.d/` into the current working directory, which defaults to `/` in a container. `HOME` is set to the clone dir for the same reason. Volumes are derived the same way `RunContainer` does for containerized DLE (reuse `getMountVolumes`/`buildVolumesFromMountPoints`), never a naive host-path bind. Engine waits for exit, captures exit code + log tail.
- Both images (`pgUpgradeImage`, target image) are pulled via `docker.PrepareImage` *before* the clone is stopped — avoids multi-GB pulls inside the downtime window that look like a hang.
- Upgrade image config: new `provision.pgUpgradeImage` option; empty is valid (feature unconfigured) — validated at the endpoint, not in `IsValidConfig`, so existing instances start unchanged. `provision.Config` is unexported inside the provisioner, so `Provisioner` gets a `PgUpgradeImage()` accessor for the handler.
- Concurrency/pre-checks: clone status must be `OK`; the srv handler rejects the request while an observation session is running (`Observer.GetObservingClone`); `UPGRADING` guards re-entry, and destroy/reset refuse `UPGRADING` clones. Dependent ZFS snapshots are immutable and unaffected by in-place changes — no revision bump (unlike reset).
- New status code `UPGRADING` + `CloneMessageUpgrading`; `StatusWarning` gains clone-level meaning and a `CloneMessageWarning`-style message set. Webhook event `clone_upgrade`; telemetry event mirroring `CloneResetEvent` (`internal/telemetry/telemetry.go`).
- HTTP response mirrors the reset handler (`routes.go:1010-1038`): 200 with empty body; result observed via clone status (deliberate consistency over 202-with-body).
- Client-side waiting: `Client.UpgradeClone` passes a ctx with a deadline to `watchCloneStatus`, which then ignores `requestTimeout` for the poll loop. No new timeout field on `Client`.
- `prepareDB` is not re-run (pg_upgrade preserves roles/databases, so the ephemeral user survives).
- Physical mode: the sync instance and pool are untouched — upgrade acts only on clone datasets. The engine-wide `dockerImage` must still match the source major for physical restores (`config.example.*.yml:44-45`); this feature does not change that.

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): engine code, upgrade image + CI, CLI, UI, docs in this repo
- **Post-Completion** (no checkboxes): image publishing to Docker Hub, manual e2e on the local instance, platform/docs-site updates, snapshot-level image resolution follow-up

## Implementation Steps

### Task 1: Models and API request types

**Files:**
- Modify: `engine/pkg/models/status.go`
- Modify: `engine/pkg/models/clone.go`
- Modify: `engine/pkg/client/dblabapi/types/clone.go`
- Modify: `engine/pkg/models/clone_test.go`

- [x] add `StatusUpgrading StatusCode = "UPGRADING"` to `status.go`, plus `CloneMessageUpgrading` and a clone-level warning message constant next to the existing clone messages (`StatusWarning` already exists)
- [x] add `DockerImage string \`json:"dockerImage,omitempty"\`` to `models.Clone` (empty = engine default image)
- [x] add `DBVersion string \`json:"dbVersion,omitempty"\`` to `models.Clone` — the major the clone is actually running; without it the API exposes no clone version at all (`DetectDBVersion` is instance-level and reads the pool data dir)
- [x] add `CloneUpgradeRequest{TargetVersion int, DockerImage string}` to `types/clone.go` with godoc
- [x] write tests for `Clone` JSON round-trip including both new fields (present/omitted), plus `CloneView` embedding
- [x] run tests - must pass before task 2

### Task 2: pg-upgrade image and upgrade script

**Files:**
- Create: `engine/Dockerfile.pg-upgrade`
- Create: `engine/scripts/pg_upgrade_clone.sh`
- Modify: `engine/.gitlab-ci.yml`

- [x] **first, verify binary compatibility**: confirmed via the image config blobs — `extended-postgres` installs PostgreSQL from `apt.postgresql.org` (`PG_VERSION=16.14-1.pgdg13+1`, `17.10-1.pgdg13+1`, PGDG key `B97B0AFC…`) on top of the official postgres image lineage, with binaries at `/usr/lib/postgresql/<major>/bin` and postgres as uid 999. Old bindirs can therefore come from the PGDG packages of the same Debian suite; the multi-stage fallback is not needed. Preserving the glibc suffix on the target tag keeps that suite consistent
- [x] write `pg_upgrade_clone.sh` (thin: no parameter validation, everything arrives via env from the engine): write `upgrade.stage` at each transition, initdb `data_new` with the provided encoding/locale/provider/checksum params, copy `postgresql.auto.conf` filtered through `<new-bindir>/postgres -C`, run `pg_upgrade --check` then `--link` (old-side `shared_preload_libraries` via `-o`), swap dirs, write logs to `upgrade_logs/`
- [x] implement the exit contract in the script: `0` success, `10` pre-transfer failure (and remove `data_new`), `20` failure at or after transfer (remove nothing). A trap maps any unexpected failure to the code implied by the current stage
- [x] write `Dockerfile.pg-upgrade`: FROM `postgresai/extended-postgres:<new>` build-arg base, add the PGDG server package for the four preceding majors, embed the script as entrypoint; one image per target major, tagged to mirror the base tag
- [x] add dedicated CI jobs in `engine/.gitlab-ci.yml` (`.pg_upgrade_job_template`, no artifacts) with a `parallel: matrix` over `PG_MAJOR`; feature/master push to the GitLab registry, release tags push `postgresai/pg-upgrade` to Docker Hub. `scripts/ci_docker_build_push.sh` gained `BUILD_ARGS` passthrough and a pg-upgrade smoke test
- [x] this task ships no Go code; Go-side tests for env construction, exit-code classification and stage mapping land in task 4 (the shell script is the explicit test exception per Testing Strategy)
- [x] verify the image builds locally for one target major and document the build command in the MR description — built `postgresai/pg-upgrade:17` from `extended-postgres:17` (old bindirs 13-16), and exercised the script end to end inside it:
  - 16 → 17 upgrade succeeded: exit 0, stage `done`, `PG_VERSION` 17, 1000 rows readable, `data_old` present
  - `postgresql.auto.conf` filtering confirmed with `old_snapshot_threshold` (valid in 16, removed in 17): dropped with a log line while `work_mem` carried over, and the upgraded cluster started. Without the filter the new server would refuse to start
  - pre-transfer failure confirmed: a mismatched `OLD_VERSION` failed at stage `check` with exit 10, `data_new` removed, and the old cluster still started with all rows
- [x] run engine tests - must pass before task 3

### Task 3: Provision config and per-clone image override

**Files:**
- Modify: `engine/internal/provision/mode_local.go`
- Modify: `engine/internal/provision/mode_local_test.go`

- [x] add `PgUpgradeImage string \`yaml:"pgUpgradeImage"\`` to `provision.Config` (no `IsValidConfig` change — empty means feature unconfigured)
- [x] add a `Provisioner.PgUpgradeImage()` accessor — `p.config` is unexported, and the HTTP handler needs to reject unconfigured instances
- [x] apply per-clone image override through the pure `resolveCloneImage(defaultImage, cloneImage)` in `StartSession` (no new `getAppConfig` parameter); it is the single resolution point that task 4 reuses
- [x] ensure `ResetSession` always uses the engine default image, with a comment recording why; callers clear `Clone.DockerImage`/`Clone.DBVersion` in task 5
- [x] write tests for config parsing with and without `pgUpgradeImage`, including that `IsValidConfig` still passes when it is unset
- [x] write tests for image override selection logic (override set / empty / registry path)
- [x] run tests - must pass before task 4

### Task 4: Provisioner upgrade orchestration

**Files:**
- Create: `engine/internal/provision/upgrade.go`
- Create: `engine/internal/provision/upgrade_test.go`
- Modify: `engine/internal/provision/docker/docker.go`

- [x] define the internal `UpgradeRequest` struct (target version, target image, encoding/locale/provider/checksum pre-flight data) and an `upgradeStage` type with the ordered stages (`prepare`, `initdb`, `check`, `transfer`, `swap`, `done`)
- [x] implement the state/stage files: engine writes `upgrade.state.json` (old major, target major, old image, target image, origin snapshot ID) and seeds `upgrade.stage` with `prepare`; both removed on success, `upgrade_logs/` kept
- [x] implement the pure classifier, named `classifyOutcome(exitCode, stage)` so exit 0 maps to `recoveryComplete` alongside the failure codes; `classifyStage` is the stage-only variant the restart path uses returning `rollbackInPlace` / `reprovision`: exit `10` → rollback, exit `20` → reprovision, any other code → decide by stage, unknown/unreadable stage → reprovision. This is the function that keeps a `--link` upgrade from restarting a half-converted cluster
- [x] add `docker.RunUpgradeContainer`: one-off container from `pgUpgradeImage` with `--user` resolved by **stat'ing the clone data directory** (`dataDirOwner`) rather than assuming an account, `--workdir <clone dir>`, `HOME` inside the clone dir, volumes derived via the existing `getMountVolumes`/`buildVolumesFromMountPoints` path (containerized DLE support), env params, wait for exit, return exit code + captured output
- [x] implement `Provisioner.UpgradeSession(session, clone, req)` in `upgrade.go`: read old major from `PG_VERSION`, validate target, `docker.PrepareImage` for both upgrade and target images *before* stopping, graceful `pg_ctl stop` (reuse/adapt `tools.StopPostgres`) then remove the container, run the upgrade container, `pgconfig.NewCorrector(newDataDir)` + `appConfig.SetExtraConf(session.ExtraConfig)`, start container with target image, delete `data_old` after successful start, clear state/stage files
- [x] implement recovery driven by the classifier: `rollbackInPlace` → remove `data_new`, restart the old image, return an error flagged as *clone healthy on old version*; `reprovision` → re-provision from the origin snapshot on the original image (`ResetSession` path), return an error flagged as *data since cloning lost*. Both carry the pg_upgrade log path
- [x] implement `Provisioner.RecoverUpgrade(session, clone)` (state is read from the clone dir) used by the startup path: same classifier over the persisted stage, plus the `done` case (adopt the new data dir, start on the target image, delete `data_old`)
- [x] write tests for version validation (equal/lower/higher, jump > 4, legacy dotted versions)
- [x] write tests for `classifyOutcome`/`classifyStage` across every exit code × stage combination, including unknown code and missing stage file
- [x] write tests for pure helpers: env construction (locale provider/ICU/old-server-options/data subdir), state and stage file round-trips, log-tail truncation, shell quoting, and that the env never uses the real `LC_COLLATE`/`LC_CTYPE` names
- [x] run tests - must pass before task 5

### Task 5: Cloning orchestration — `UpgradeClone`, recovery and safety guards

**Files:**
- Modify: `engine/internal/cloning/base.go`
- Modify: `engine/internal/cloning/storage.go`
- Modify: `engine/internal/cloning/base_test.go`
- Modify: `engine/internal/webhooks/events.go`
- Modify: `engine/internal/telemetry/telemetry.go`

- [x] implement `Base.UpgradeClone(cloneID, req)` in a new `internal/cloning/upgrade.go` (base.go is already past the size guidance) mirroring `ResetClone`: pre-checks (wrapper/session exist, status `OK`), pre-flight query for encoding/locale/provider/checksums/tablespaces via `ConnectToClone`, set status `UPGRADING`, async goroutine → `provision.UpgradeSession`
- [x] on success: set `Clone.DockerImage` + `Clone.DBVersion`, status `OK` with "upgraded to PostgreSQL <N>" message, `SaveClonesState()`, emit webhook `clone_upgrade` (`CloneUpgradeEvent` const) + telemetry event
- [x] on in-place rollback: status `WARNING`, clone still on the old major, message = cause + pg_upgrade log path
- [x] on re-provision: status `WARNING`, message stating the clone was restored from snapshot `<id>` and writes made since cloning are lost, + log path. Only if the re-provision itself fails → `FATAL`
- [x] never leave an upgraded/failed clone `FATAL` with no container — `filterRunningClones` deletes `FATAL` wrappers and `cleanupInvalidClones` then destroys their datasets
- [x] clear `Clone.DockerImage`/`Clone.DBVersion` in `ResetClone` so reset returns the clone to the engine default version
- [x] guards: exempt `StatusUpgrading` in `isIdleClone` (next to `StatusExporting`), refuse `DestroyClone` (`destroyPreChecks`) and `ResetClone` while `UPGRADING`
- [x] implement `Base.RecoverInterruptedUpgrades()` and call it in `Run` immediately after `RestoreClonesState()` and before `restartCloneContainers` — for every clone persisted as `UPGRADING`, read `upgrade.state.json` + `upgrade.stage` and delegate to `provision.RecoverUpgrade`. Because it runs before the filters, no changes are needed in `filterRunningClones`/`cleanupInvalidClones`
- [x] write tests for pre-check failures (not found, not started, wrong status), destroy/reset refusal while `UPGRADING`, the `isIdleClone` exemption, the recovery selection, locale-provider/ICU column mapping across majors, status mapping per outcome, and image/version bookkeeping per outcome — using the existing `BaseCloningSuite` style (no provisioner mock; async success/failure transitions are covered by e2e, not unit tests)
- [x] write tests for the recovery decision over persisted state (each stage → expected action), keeping the decision itself a pure function
- [x] run tests - must pass before task 6

### Task 6: HTTP API endpoint, client, swagger

**Files:**
- Modify: `engine/internal/srv/routes.go`
- Modify: `engine/internal/srv/routes_test.go`
- Modify: `engine/internal/srv/server.go`
- Modify: `engine/pkg/client/dblabapi/clone.go`
- Modify: `engine/pkg/client/dblabapi/clone_test.go`
- Modify: `engine/api/swagger-spec/dblab_server_swagger.yaml`
- Modify: `engine/api/swagger-spec/dblab_openapi.yaml`

- [x] add `upgradeClone` handler in a new `internal/srv/upgrade.go` (keeps routes.go from growing): decode `CloneUpgradeRequest`, extract a pure validation/resolution helper, reject while an observation session is running for the clone (`s.Observer.GetObservingClone`), call `Cloning.UpgradeClone`, respond 200 empty like the reset handler
- [x] the resolution helper: explicit `dockerImage` wins; otherwise substitute the major in the current image tag preserving suffixes. Split on the last `/` then the last `:` so registry hosts with ports (`registry:5000/img:16`) and SE repos (`registry.gitlab.com/postgres-ai/se-images/<provider>:16-...`) resolve correctly; hard-fail on a non-numeric tag (`:latest`) with "specify dockerImage explicitly"
- [x] reject when `pgUpgradeImage` is unconfigured (via the new `Provisioner.PgUpgradeImage()` accessor), when the target has no `configs/standard/postgres/default/<major>` directory, and when the jump exceeds the four majors the image covers
- [x] register `POST /clone/{id}/upgrade` in `server.go` next to `/clone/{id}/reset`
- [x] add `Client.UpgradeClone` and `Client.UpgradeCloneAsync` to `dblabapi/clone.go`. Extracted a shared `postCloneAction` helper (the `dupl` linter rejected a third copy of the POST-and-discard block); `ResetClone`/`ResetCloneAsync` now use it too. The sync variant passes a ctx **with a deadline** to `watchCloneStatus`, which then bypasses `requestTimeout` entirely — no new client timeout field. Treat `OK` as success and `WARNING`/`FATAL` as failure, surfacing the status message
- [x] document the endpoint in both swagger specs and add `dockerImage` + `dbVersion` to the `Clone` schema, and `UPGRADING`/`WARNING` to the documented status codes
- [x] write tests for the pure validation/resolution helper (tag substitution incl. suffixed tags, registry host with port, SE repo path, `:latest`, explicit image, unconfigured feature, bad target version, unsupported jump, unsupported major) in `routes_test.go`
- [x] write tests for the client methods (success, `WARNING`, async, and that a caller deadline ends the wait) in `clone_test.go`
- [x] run tests - must pass before task 7

### Task 7: CLI command

**Files:**
- Modify: `engine/cmd/cli/commands/clone/actions.go`
- Modify: `engine/cmd/cli/commands/clone/command_list.go`
- Create: `engine/cmd/cli/commands/clone/actions_test.go`

- [x] add `dblab clone upgrade CLONE_ID --target-version N [--docker-image IMG] [--async]` following the reset command's structure and output style
- [x] the sync path sets a ctx deadline appropriate for an upgrade; document that the existing global `--request-timeout` flag (`cmd/cli/main.go:68`) still bounds individual HTTP requests
- [x] the failed-upgrade error carries the clone status message verbatim, which already names the version the clone ended on and the log location; no reset hint is added because recovery is now automatic (rolled back / restored from snapshot) and including the log path
- [x] write tests for the flag-to-`CloneUpgradeRequest` mapping, plus that the subcommand is registered with a required target version
- [x] run tests - must pass before task 8

### Task 8: CE UI

**Files:**
- Create: `ui/packages/ce/src/api/clones/upgradeClone.ts`
- Create: `ui/packages/shared/types/api/endpoints/upgradeClone.ts`
- Modify: `ui/packages/shared/types/api/entities/clone.ts`
- Modify: `ui/packages/shared/utils/clone.ts`
- Modify: `ui/packages/shared/pages/Clone/stores/Main.ts`
- Modify: `ui/packages/shared/pages/Clone/index.tsx`
- Modify: `ui/packages/ce/src/App/Instance/Clones/Clone/index.tsx`

- [x] add `upgradeClone` API binding mirroring `resetClone.ts`, and declare it **optional** on the shared `Api` type (`upgradeClone?:`, like `resetClone?:`) so the closed Platform app keeps building
- [x] extend `CloneDto` with `dockerImage?` and `dbVersion?`, and add `UPGRADING` + `WARNING` to the status union
- [x] add `UPGRADING: 'waiting'` and `WARNING: 'warning'` to `STATUS_CODE_TO_TYPE` (the `Status` component already supports `'warning'`, `components/Status/index.tsx:16`)
- [x] add `WARNING` to `checkIsCloneStable` — it is a terminal state, and without it the clone page and create-clone page poll forever
- [x] show `dbVersion` on the clone page so the result of an upgrade is visible
- [x] add an "Upgrade" action on the clone page (new `UpgradeCloneModal`, hidden via the store's `isUpgradeSupported` when the host app does not provide the endpoint): dialog with target major version input, confirmation noting that reset rolls the upgrade back and that a failed upgrade may restore the clone from its snapshot; hide the action when the api method is not provided
- [x] verify types compile in the touched packages — `tsc --noEmit` clean for both `shared` and `ce` against the existing node_modules; build/tests left to CI
- [x] run engine tests (unchanged) - must pass before task 9

### Task 9: Config examples and docs

**Files:**
- Modify: `engine/configs/config.example.logical_generic.yml` (and the other `config.example.*.yml`)
- Modify: `README.md`

- [x] add commented `pgUpgradeImage` directly under `provision:` (not inside the `databaseContainer: &db_container` anchor that `provision:` merges — the anchor is shared with retrieval containers) with a one-line explanation
- [x] add `clone_upgrade` to the webhook trigger-event lists in the config examples
- [x] add a README section: what clone major upgrade does, `--link` semantics and why rollback is stage-dependent, reset-as-rollback, the `WARNING` terminal state, extension caveats, non-default tablespaces unsupported, target majors limited to those with a `configs/standard/postgres/default/<major>` directory and within four majors of the current one, iteration-1 limitation (clones from upgraded snapshots), physical-mode note (sync instance untouched)
- [x] verify example configs still pass config parsing tests (and that the new key stays commented, so nothing changes for existing instances)
- [x] run tests - must pass before task 10

### Task 10: Verify acceptance criteria
- [x] verify all requirements from Overview are implemented (API + CLI + UI + image + config + docs + webhook + swagger)
- [x] verify edge cases (covered by unit tests; the interrupted-upgrade stages are now expressed as data-directory facts rather than stages): unconfigured `pgUpgradeImage`, target ≤ current, jump > 4 majors, major without a default config dir, non-default tablespace present, clone busy/observed, pre-transfer failure with in-place rollback, post-transfer failure with re-provision, reset after upgrade, destroy/reset refused mid-upgrade, engine restart with an upgraded clone (container reused), engine restart during an upgrade at each stage (`prepare`/`check` → rollback, `transfer` → re-provision, `done` → completion)
- [x] run full test suite: `cd engine && make test` — 52 packages, no failures
- [x] run linter: `cd engine && make run-lint` — 0 issues
- [x] run `cd engine && make fmt` and check for diffs — clean

### Task 11: [Final] Update documentation
- [x] update README.md if anything shifted during implementation
- [x] update CLAUDE.md if new patterns were established (per-clone image override, stage-gated upgrade recovery, `UPGRADING`/`WARNING` clone statuses, startup recovery ordering in `Base.Run`)
- [x] move this plan to `docs/plans/completed/`

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- e2e on the local test instance (container `dblab_server_local`, API :12345): create clone on PG 16 → upgrade to 17 → connect and verify `SELECT version()` → verify managed config applied (external connection works, `pg_hba` correct) → verify `dbVersion` in the API → reset and verify it is back on 16 (user builds/pulls the pg-upgrade test image themselves)
- pre-transfer failure path: point `pgUpgradeImage` at an image lacking the needed old-major bindir; confirm exit `10`, in-place rollback, clone running on the old version, status `WARNING` referencing the log
- post-transfer failure path: interrupt the upgrade container during `transfer` (kill it); confirm exit is classified unsafe, the clone is re-provisioned from its snapshot, and the old cluster is never started
- engine-restart paths: restart DLE with a clone stuck at each stage (`prepare`, `check`, `transfer`, `done`) and confirm the recovery action per stage, and that no dataset is destroyed by `cleanupInvalidClones`
- containerized-DLE check: run the upgrade with the engine itself in a container (volume-derivation path, uid resolution)

**External system updates**:
- publish `postgresai/pg-upgrade:<major>-<extver>` images to Docker Hub (CI builds and pushes to the GitLab registry; Docker Hub credentials are not wired into this project's CI)
- Platform Console UI (closed repo at ~/work/postgres-ai/platform-all) may adopt the same upgrade action; the shared `Api` field is optional so nothing breaks meanwhile
- docs site (postgres.ai docs repo) needs a how-to page for the feature

**Follow-up (iteration 2)**:
- snapshot-level version/image resolution in `StartSession` so clones created from a committed upgraded snapshot start on the right image (record major/image on the snapshot or detect `PG_VERSION` in the snapshot data dir)
- optional post-upgrade tasks (`vacuumdb --analyze-in-stages`, `ALTER EXTENSION ... UPDATE`) as a config- or request-level opt-in
- non-default tablespace support (requires mounting tablespace locations into the upgrade container and into clone containers)

**Known risks to watch during implementation**:
- old-bindir binary compatibility (Task 2, first checkbox) is the load-bearing assumption of the whole image design — verify before writing the Dockerfile
- pg_upgrade's loadable-library check requires new-major `.so`s for every extension installed in the source DB; deriving the upgrade image from `extended-postgres:<new>` matches the set by construction, but source DBs with extensions outside that set will fail at `--check` (a pre-transfer failure, so the clone rolls back cleanly) — the log path in the status message is the user's diagnostic
- locale/encoding/locale-provider mismatches between initdb defaults and the source cluster are the top pg_upgrade failure mode; the pre-flight query must run before the clone container is stopped
- old-side `shared_preload_libraries` must be handled for pg_upgrade's own old-server start (via `-o`), and the new data dir must get managed config re-applied before first start or the clone boots unreachable
- glibc/collation: never change the glibc build across an upgrade (tag substitution preserves the suffix); pg_upgrade does not reindex
- `upgrade_logs/` stays inside the clone dataset, so a snapshot committed from an upgraded clone carries it into child clones; the state/stage files are removed on success so they do not

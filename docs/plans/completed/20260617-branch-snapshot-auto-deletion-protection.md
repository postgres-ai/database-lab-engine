# Branch & Snapshot Auto-Deletion and Protection

## Overview

Extend the existing clone-level protection/auto-deletion mechanism to **branches** and **snapshots**:

- **Deletion protection** — mark a branch or snapshot as protected so it cannot be deleted (manually or automatically).
- **Time-based auto-deletion** — automatically delete a branch or snapshot after N minutes of being *unused* (no dependent clones and no dependent child snapshots/branches), unless it is protected.

This implements bullets 1–2 of work item
[#627](https://gitlab.com/postgres-ai/database-lab/-/work_items/627). It is a corrected re-implementation of the
closed draft [MR !1071](https://gitlab.com/postgres-ai/database-lab/-/merge_requests/1071), which was rejected in review
for fundamental design problems (see "Review-driven design constraints" below).

### Key benefits
- Prevents accidental loss of important branches/snapshots (protection).
- Reclaims storage automatically from abandoned branches/snapshots (auto-deletion), like `maxIdleMinutes` does for clones.
- Reuses ZFS as the single source of truth — no parallel JSON state to drift or corrupt.

## Review-driven design constraints (from MR !1071)

These are non-negotiable corrections derived directly from the reviewer's (akartasov) objections to MR !1071:

1. **ZFS is the source of truth — no JSON entity store.** Snapshot/branch metadata lives in ZFS user properties
   (DLE already stores `dle:branch`, `dle:parent`, `dle:root`, `dle:message`, `dblab:datastateat` this way). Unlike
   clones, branches/snapshots are *not* tied to containers, so they need no persisted session state.
2. **Multi-pool aware.** All logic iterates every pool via `pool.Manager.GetFSManagerList()`; branch metadata is
   anchored to a pool-scoped dataset.
3. **Stable anchor for branch metadata.** Branch protection/delete-at is stored on the **branch's own dataset**
   (`<pool>/branch/<branchName>`), which is stable for the branch lifetime and inherently pool-scoped — it does not
   move as the branch head advances. (Reviewer: "the label changes over time... better to stick to the initiating
   snapshot... take into account the dataset.") *Open point to confirm in review: branch dataset vs. initiating
   snapshot as the anchor — both satisfy "stable + pool-scoped"; the branch dataset is 1:1 with the branch and avoids
   name-keying a shared fork-point snapshot.*
4. **No new snapshot struct.** Reuse existing `models.Snapshot` / `thinclones.SnapshotProperties`; do not add a third
   snapshot representation.
5. **Single `error` returns.** No `(bool, string)` / `(bool, string, error)` combos; no `select` to read a single
   error; no error-type switch where the action is identical.
6. **Pack mutable params into one request struct.** PATCH handlers take one input struct, not positional flags.
7. **No legacy time formats.** New code uses the existing `*models.LocalTime` / RFC3339; no "legacy format" branch.

## Design decisions (confirmed with user)

- **Auto-deletion is "safe-only".** The background sweeper deletes a branch/snapshot **only when it has zero
  dependents**. It never force-destroys dependents in the background. Recursive/force destroy stays a manual,
  explicit operation (existing `force` flag on snapshot delete / `-R` on branch dataset destroy). This drops the
  tri-state `AutoDeleteMode` (Off/Soft/Force) from MR !1071.
- **Protection and scheduled deletion are mutually exclusive.** Protecting an entity clears any `deleteAt`; setting an
  explicit `deleteAt` (or the sweeper scheduling one) only happens on unprotected entities. Protection mirrors clones:
  optional timed protection via `ProtectedTill` (nil = indefinite).
- **Scope:** snapshots + branches, engine + API client + CLI + config + swagger. **UI is out of scope** (follow-up).
- **Automatic refresh snapshots are excluded** from time-based auto-deletion (they remain governed by the existing
  count-based `CleanupSnapshots` retention). Auto-deletion targets user-created branch snapshots/commits.

## Context (from discovery)

**Files/components involved:**
- ZFS property layer: `engine/internal/provision/thinclones/zfs/branching.go` (`setProperty`/`getProperty`,
  `GetSnapshotProperties`, `GetRepo`/`GetAllRepo`, `listBranches`, `ListAllBranches`), `zfs/zfs.go`
  (`listDetails` `defaultFields`, `CleanupSnapshots` at ~line 581, `DestroySnapshot`, `DestroyDataset`),
  `zfs/snapshots_filter.go`.
- Consumer interface: `engine/internal/provision/pool/manager.go` (`Branching`, `Snapshotter` interfaces),
  `pool/pool_manager.go` (`GetFSManagerList`, `GetFSManager`).
- Models: `engine/pkg/models/snapshot.go`, `engine/pkg/models/branch.go`, `engine/pkg/models/clone.go`
  (`IsProtected`/`ProtectionExpiresIn` pattern), `engine/pkg/models/local_time.go`.
- HTTP server: `engine/internal/srv/server.go` (route registration ~230–253), `engine/internal/srv/routes.go`
  (`deleteSnapshot` ~216, `patchClone` ~718), `engine/internal/srv/branch.go` (`deleteBranch` ~540,
  `listBranches`, `destroyBranchDataset`).
- Clone pattern to mirror: `engine/internal/cloning/base.go` (`runIdleCheck`/`runProtectionLeaseCheck` ~113,
  `UpdateClone` ~461, `calculateProtectionTime` ~489, `isIdleClone` ~835, `destroyPreChecks` ~340),
  `engine/internal/cloning/snapshots.go` (clone counting: `GetCloneNumber`, `counterClones`),
  `engine/internal/cloning/storage.go` (why clones persist — NOT needed here).
- API client/types: `engine/pkg/client/dblabapi/branch.go`, `snapshot.go`,
  `engine/pkg/client/dblabapi/types/clone.go` (`CloneUpdateRequest` pattern at ~20).
- CLI: `engine/cmd/cli/commands/branch/`, `engine/cmd/cli/commands/snapshot/`,
  `engine/cmd/cli/commands/clone/actions.go` (`parseProtectedFlag`/`parseDurationMinutes` to reuse).
- Config: `engine/internal/cloning/base.go` `Config` (`maxIdleMinutes`, `protection*`), `engine/configs/*.yml`,
  wiring in `engine/cmd/database-lab/main.go`.
- Telemetry/webhooks: `engine/internal/telemetry/{telemetry.go,events.go}`, `engine/internal/webhooks/events.go`.
- Swagger: `engine/api/swagger-spec/dblab_openapi.yaml`.

**Existing ZFS user properties:** `dle:branch`, `dle:parent`, `dle:child`, `dle:root`, `dle:message`,
`dblab:datastateat`, `dblab:isroughdsa`. We add `dle:protected_till` and `dle:delete_at`.

## Development Approach
- **Testing approach:** Regular (code first, then table-driven tests) per existing codebase convention.
- Complete each task fully (incl. tests passing) before starting the next.
- Make small, focused changes; preserve existing patterns; avoid `else`/nested-if chains; single-`error` returns.
- **Every task includes new/updated tests** (success + error/edge cases). Tests are required, not optional.
- Run `cd engine && make test` and `make run-lint` after each task; fix before proceeding.
- Maintain backward compatibility: new model fields are additive and `omitempty`; absent ZFS props mean
  "unprotected / no scheduled deletion".

## Testing Strategy
- **Unit tests** (required every task): model helpers, ZFS property parsing, the sweeper decision function (pure
  logic, isolated from ZFS), PATCH request validation/mutual-exclusivity, API client round-trips
  (mirror `clone_test.go`), CLI flag parsing.
- **ZFS-dependent methods:** follow the existing approach in the `zfs` package (the property setters are thin command
  wrappers; test the command-string construction and the parsing of `zfs list` output, not a live pool).
- **e2e:** no UI in scope. Engine integration is covered by `engine/test/*.sh`; note any manual `dblab` CLI scenario
  in Post-Completion rather than adding new shell e2e here unless a maintainer requests it.

## Progress Tracking
- Mark completed items `[x]` immediately. Add discovered tasks with ➕. Flag blockers with ⚠️.
- Keep this file in sync if scope changes.

## What Goes Where
- **Implementation Steps** (checkboxes): code, tests, config examples, swagger, README — all in this repo.
- **Post-Completion** (no checkboxes): manual CLI verification on a live ZFS instance, UI follow-up, telemetry
  dashboard updates.

## Implementation Steps

### Task 1: Add protection / delete-at fields to branch & snapshot models

**Files:**
- Modify: `engine/pkg/models/snapshot.go`
- Modify: `engine/pkg/models/branch.go`
- Create: `engine/pkg/models/snapshot_test.go` (if absent) / add to existing model test file
- Create: `engine/pkg/models/branch_test.go` (if absent)

- [x] add `Protected bool`, `ProtectedTill *LocalTime` (`omitempty`), `DeleteAt *LocalTime` (`omitempty`) to **`Snapshot`** only — `SnapshotView` embeds `*Snapshot`, fields surface automatically (snapshot.go)
- [x] for branches, added the three fields to **`BranchView`** and **`SnapshotDetails`** (branch.go); left the bare `Branch` struct untouched
- [x] added shared free functions `isProtected`/`protectionExpiresIn` in `models/protection.go`; `Clone`, `Snapshot`, and `BranchView` all delegate to them (no interface)
- [x] mutual-exclusivity invariant documented in plan ("Mutual exclusivity") and enforced by the Task-5 PATCH handlers (the structs cannot encode it; ➕ noted)
- [x] tests in `snapshot_test.go` / `branch_test.go` (unprotected, indefinite, timed-future, timed-expired) + `ProtectionExpiresIn`
- [x] `go test ./pkg/models/` passes

### Task 2: ZFS property constants + get/set for protection & delete-at

**Files:**
- Modify: `engine/internal/provision/thinclones/zfs/branching.go`
- Modify: `engine/internal/provision/thinclones/manager.go` (the `thinclones.SnapshotProperties` struct)
- Modify: `engine/internal/provision/pool/manager.go` (`Branching` interface)
- Modify: `engine/internal/provision/thinclones/zfs/branching_test.go` (or nearest existing test)

- [x] added constants `protectedTillProp`, `deleteAtProp`, and `foreverProtection` sentinel
- [x] added `SetProtectedTill`/`SetDeleteAt` on `setProperty` (empty clears to `-`); no `zfs inherit`
- [x] timestamps stored as UTC RFC3339 with `Z` (round-trip test proves no `-`-trim corruption)
- [x] setters take a `target` that is either a snapshot or a branch dataset — single methods
- [x] source-filtered local reads (review hardening): `GetProtection(target) (thinclones.ProtectionProperties, error)` — single entity, one `zfs get -s local property,value` call; and `ListProtection() (map[string]…)` — all snapshots in the pool, one `zfs get -s local -t snapshot -r` call. These are the authoritative readers Tasks 4/6 + BranchView reuse (the per-property `getLocalProperty` was removed — both batch instead)
- [x] added `ProtectedTill`/`DeleteAt` fields to `thinclones.SnapshotProperties` + new `ProtectionProperties` struct
- [x] extended the `Branching` interface with `SetProtectedTill`, `SetDeleteAt`, `GetProtection`, `ListProtection`
- [x] added LVM stubs (lowercase logs) + updated both `mockFSManager`s (mode_local_test.go, pool_manager_test.go)
- [x] tests: set/clear command-string, timestamp round-trip, `GetProtection`, `ListProtection`; `recordingRunnerMock` serves the batched `-s local` form
- [x] `go test ./internal/provision/...` passes

### Task 3: Surface new properties in repo reads and model population

> ⚠️ **Do NOT add the new props to `defaultFields`/`listDetails`** (zfs.go:1094-1170). That parser uses
> whitespace `strings.Fields()` + **hardcoded positional indices** (`fields[14]`/`fields[15]`) and only tolerates
> `numberFields-1` (ONE missing custom field). Two new optional props would trip the hard error at zfs.go:1130 for
> every existing snapshot that lacks them. Use the tab-delimited `repoFields` path instead — it splits with
> `SplitN(line, "\t", len(repoFields))` and is robust.

**Files:**
- Modify: `engine/internal/provision/thinclones/zfs/branching.go` (`repoFields`, `getRepo`, `GetSnapshotProperties`)
- Modify: `engine/internal/provision/mode_local.go` (new `Provisioner.ListProtection`, skip-and-continue on per-pool error)
- Modify: `engine/internal/cloning/snapshots.go` (cached, local protection merge in `fetchSnapshots`)
- Modify: `engine/internal/cloning/base.go` (`ReloadSnapshots` refreshes the protection cache)
- Modify: `engine/pkg/models/protection.go` (shared `ParseProtectedTill`/`ParseDeleteAt`, `ProtectionForever`)
- Modify: `engine/internal/provision/thinclones/zfs/zfs_test.go` (fixtures + `TestGetRepoProtection`)
- (deferred) `engine/internal/srv/branch.go` branch-view assembly → Task 5

- [x] appended `protectedTillProp` and `deleteAtProp` to `repoFields` (feeds both `getRepo` and `GetSnapshotProperties`)
- [x] parse the two new tab fields in `getRepo` (→ `SnapshotDetails`, via `models.ParseProtectedTill`/`ParseDeleteAt`; malformed → unprotected + warn, never silent permanent protection) and `GetSnapshotProperties` (→ raw `SnapshotProperties` strings)
- [x] populate `Snapshot.Protected/ProtectedTill/DeleteAt` from a **cached, LOCAL** protection map (`Provisioner.ListProtection`, batched `zfs get -s local`) merged by snapshot ID in `fetchSnapshots`; cache is lazy-loaded and refreshed by `ReloadSnapshots`, so reads do not run a ZFS call per request, and the list reflects each snapshot's own protection (no inherited branch-dataset value)
- [x] ✅ **(done in Task 5)** branch-dataset property read for `BranchView` display — `listBranches` populates `Protected/ProtectedTill/DeleteAt` per branch via `getFSManagerForSnapshot` + `GetProtection(<pool>/branch/<name>)`.
- [x] tests: `getRepo`/`GetSnapshotProperties` parsing with new columns present AND empty (`-`); added `TestGetRepoProtection` (forever/timed/scheduled-deletion)
- [x] `go test ./internal/provision/... ./internal/cloning/...` passes; full `make test` + `make run-lint` (0 issues) green

### Task 4: Enforce protection in delete paths + count-based cleanup; extract reusable destroy helpers

**Files:**
- Modify: `engine/internal/srv/routes.go` (`deleteSnapshot` → thin handler + `destroySnapshotByID` and helpers)
- Modify: `engine/internal/srv/branch.go` (`deleteBranch` → thin handler + `destroyBranchByName`; removed `destroyBranchDataset`)
- Modify: `engine/internal/provision/thinclones/zfs/zfs.go` (`DestroyBranchDataset`, `getProtectedSnapshots`, `excludeBusySnapshots`, `CleanupSnapshots`)
- Modify: `engine/internal/cloning/base.go` (`WithBranchDeletionLock`, `cloneDependentSnapshotLocked`)
- Modify: `engine/pkg/models/protection.go` (`ProtectedTillActive`)
- Modify: interface + LVM stub + both mocks (`DestroyBranchDataset`)
- Tests: `branching_test.go`, `base_test.go`, `protection_test.go`

- [x] `destroySnapshotByID` refuses a protected snapshot (`ProtectedTillActive`) before dependency checks and regardless of `force` — so protection blocks manual, force, and sweeper deletes
- [x] `destroyBranchByName` refuses a protected branch (reads the branch dataset's local `GetProtection`)
- [x] extracted `destroySnapshotByID(snapshotID, force) error` (handler is now a thin wrapper) — preserves the `ParseCloneName`-skip (in `dependentClones`), keeps native `clones` property + non-force `zfs destroy`; split into `dependentClones`/`relabelParentBranch`/`cleanupSnapshotDataset`/`cleanupParentBranchLabels` to keep functions small
- [x] `destroyBranchByName` runs `WithBranchDeletionLock(toRemove, fn)` → `DestroyBranchDataset` (live child-fs check, then `-R`); never the raw `DestroyDataset(-R)`. Replaced the old non-atomic `GetCloneNumber` precheck; events moved to the handler
- [x] `DestroyBranchDataset` (zfs): `zfs list -H -r -t filesystem -o name`; refuse if any descendant other than the dataset itself; else `DestroyDataset`. Added to `Branching` interface + LVM stub + mocks
- [x] `WithBranchDeletionLock` (cloning/base.go): holds `cloneMutex` and checks the **clones map** (`cloneDependentSnapshotLocked`) for any clone of the branch's snapshots, then `destroy()` under the lock — atomic w.r.t. the clone *registration* (`setWrapper`, under `cloneMutex`) that precedes the unlocked `zfs clone`. Review fix: the map is checked, not `NumClones`, because the counter is bumped under `snapshotMutex` at a slightly later point than registration (a clone could be registered but uncounted); this also lets the lock be `cloneMutex`-only. Protection-rule glue extracted to `ensureNotProtected` shared by both delete paths
- [x] `CleanupSnapshots`: `getProtectedSnapshots()` (local active `dle:protected_till` via `ListProtection`) appended to `busySnapshots`; `excludeBusySnapshots` now `regexp.QuoteMeta`s each name so metacharacters match literally (Risks §5)
- [x] tests: `ProtectedTillActive` (forever/future/past/malformed), `DestroyBranchDataset` (refuses on child fs / proceeds when only itself), `WithBranchDeletionLock` (refuses when clones, runs when zero — real locked behavior), `getProtectedSnapshots` filtering, `excludeBusySnapshots` metacharacter escaping. Full-handler "rejected" e2e is manual (no Server/httptest harness in srv) → Post-Completion
- [x] `make test` (no failures) + `make run-lint` (0 issues) + `make build` green
- [x] run tests — must pass before next task

### Task 4.5: Add the `retention` config struct (consumed by Tasks 5 & 6)

> Extracted ahead of Task 5 because `patchSnapshot`/`patchBranch` (Task 5) compute the protection cap from
> `retention.protectionMaxDurationMinutes`, and the sweeper (Task 6) reads the same section. Defining only the
> struct + parsing + server wiring here avoids a Task 5 that cannot compile against its stated cap source.

**Files:**
- Modify: `engine/internal/srv/config/config.go` (new `Retention` struct)
- Modify: `engine/pkg/config/config.go` (`Retention srvCfg.Retention` top-level section)
- Modify: `engine/internal/srv/server.go` (the `Server` holds the parsed `retention` config)
- Modify: `engine/cmd/database-lab/main.go` (`server.Retention = cfg.Retention`)

- [x] added `srvCfg.Retention` struct (`unusedSnapshotMinutes`, `unusedBranchMinutes`, `checkIntervalMinutes`, `protectionMaxDurationMinutes`, `maxDeletionsPerTick`), not bolted onto `cloning.Config`; all fields optional, zero ⇒ disabled
- [x] added top-level `retention:` to `pkg/config.Config`; `Server.Retention` field set in `main.go`; missing section parses to the zero value (backward compatible)
- [x] parsing test (present, absent, partial) in `srv/config/config_test.go`
- [x] `make test` + `make run-lint` (0 issues) green

### Task 5: PATCH API endpoints, request types, and client methods

**Files:**
- Modify: `engine/internal/srv/server.go` (register `PATCH /branch/{branchName}`, `PATCH /snapshot/{id:.*}`)
- Modify: `engine/internal/srv/branch.go` (add `patchBranch`)
- Modify: `engine/internal/srv/routes.go` (add `patchSnapshot`)
- Modify: `engine/pkg/client/dblabapi/types/clone.go` (or a new `types` file) — add request structs
- Modify: `engine/pkg/client/dblabapi/branch.go`, `engine/pkg/client/dblabapi/snapshot.go` (client methods)
- Create: `engine/pkg/client/dblabapi/branch_test.go`; Modify: `engine/pkg/client/dblabapi/snapshot_test.go`, `internal/srv/protection_test.go`

- [x] added `SnapshotUpdateRequest` and `BranchUpdateRequest` (`Protected *bool`, `ProtectionDurationMinutes *uint`, `DeleteAt *models.LocalTime`) — pointers give three-state, the intentional divergence from `CloneUpdateRequest`
- [x] extracted `models.CalculateProtectionTime(durationMinutes, defaultMin, maxMin)`; `cloning.calculateProtectionTime` now delegates (no duplication, no copied `else`)
- [x] `patchSnapshot`/`patchBranch` enforce mutual exclusivity via the shared `applyProtectionUpdate` (protect clears `delete_at`; `delete_at` clears protection; protect+deleteAt rejected; empty rejected), compute `ProtectedTill` with `maxMin = Server.Retention.ProtectionMaxDurationMinutes`, then call the Task-2 ZFS setters. **`patchBranch` fans the write out to every pool's `<pool>/branch/<name>`** via `branchDatasets` (iterates `GetFSManagerList` + per-pool `ListBranches`), not `s.pm.First()`
- [x] added `branchDatasets` per-pool enumeration helper (used by `patchBranch`; the Task-6 sweeper will reuse it) and `readProtection`
- [x] **(carried from Task 3)** `listBranches` now populates `BranchView.Protected/ProtectedTill/DeleteAt` via `getFSManagerForSnapshot` + `GetProtection(<pool>/branch/<name>)`
- [x] client methods `UpdateSnapshot`/`UpdateBranch`; all three `Update*` client methods now share a generic `patchJSON[T]` helper (resolves the `dupl` lint and DRYs `UpdateClone`)
- [x] tests: `applyProtectionUpdate` (mutual exclusivity, forever/duration/cap, unprotect-leaves-delete_at), client round-trips for `UpdateSnapshot`/`UpdateBranch`
- [x] `make test` (no failures) + `make run-lint` (0 issues) + `make build` green

### Task 6: Background "safe-only" auto-deletion sweeper

> **Scope guard:** this is the largest task. Keep it to the mechanical sweep (decide → schedule → delete) over the
> primitives built in Tasks 2/4/4.5. Do NOT grow a configurable policy engine, per-entity rule DSL, or new
> persistence — that is the kind of scope creep the plan set out to avoid (MR !1071).

**Files:**
- Create: `engine/internal/srv/auto_delete.go`
- Create: `engine/internal/srv/auto_delete_test.go`
- Modify: `engine/internal/srv/server.go` (start the sweeper goroutine in `Run`; the `Server` already holds the Task-4.5 `retention` config, `s.pm`, and `s.Cloning`)

- [x] **config:** consumes the Task-4.5 `retention` struct directly (`s.Retention`); `defaultRetentionInterval = 5m`, `defaultMaxDeletionsPerTick = 50`, both overridden by config when non-zero
- [x] added the pure `nextDeleteState(now, protected, hasDependents, currentDeleteAt *time.Time, retention) (*time.Time, bool)` — protected/in-use → nil schedule; first-seen-unused → now+retention; reached → delete; else keep. Fully unit-tested without ZFS
- [x] **true-leaf eligibility (CRITICAL — Risks §1):** `snapshotIsLeaf(details, branchHeads)` requires `dle:child`, `dle:root`, `dle:branch`, AND the native `clones` property all empty, plus not in the branch-head set. `HasDependentEntity`'s return value is never used. **Deviation (documented):** the branch-head set is built from `repo.Branches` (`branchHeadSet`) rather than exposing the unexported `getBranchHeadSnapshots` through the interface — the clone-origin/fork-point snapshots that method adds always carry `dle:root` (set by `createBranch`'s `SetRoot`) and are already excluded by the root check, and `_pre`/automatic snapshots are skipped, so the origin chain is provably redundant. This keeps Task 6 to its 3 planned files (no interface/LVM/mock churn)
- [x] sweep loop (`runAutoDeletion`→`runRetentionSweep`): timer at `retentionInterval()`; iterates **all pools** via `s.pm.GetFSManagerList()`; **skips any pool whose locked `fsm.Pool().Status() != ActivePool`** (re-checked per pool); `skipAutoDelete` drops automatic `pool@…` snapshots and `_pre` snapshots; snapshot vs branch sweeping each gated on its own window (`UnusedSnapshotMinutes`/`UnusedBranchMinutes`); in-use → `delete_at` cleared (via reconcile), unset → scheduled, reached → deleted via Task-4 helpers
- [x] **batch the property reads (Risks §13):** snapshots use one `fsm.ListProtection()` per pool per tick; branch protection read per-branch via `GetProtection(branchDataset)` (branch props live on the dataset, not in the `-t snapshot` listing; branches are few)
- [x] **pre-destroy protection re-read (Risks §10):** `isProtectedNow(fsm, target)` re-reads `GetProtection` immediately before every destroy and aborts if now-protected; a read error is treated as protected (fail-safe)
- [x] **refresh the cloning protection cache:** `runRetentionSweep` calls `s.Cloning.ReloadSnapshots()` exactly once at the end of a sweep, gated on a `sweepState.changed` flag set by any property write or deletion — avoids a per-write reload storm
- [x] **snapshot leaf check uses authoritative live signals (Risks §9):** scheduling uses the native ZFS `clones` property (`SnapshotDetails.Clones` from the repo); the authoritative TOCTOU backstop at deletion is the non-force `zfs destroy` inside `destroySnapshotByID`. **Deviation:** the advisory in-memory `GetCloneNumber` is intentionally NOT called in the hot loop — it logs `snapshot not found` for any snapshot absent from the box, and its registered-but-unprovisioned-clone window is already covered for branches by `WithBranchDeletionLock`'s clones-map check and for snapshots by non-force destroy
- [x] **concurrency (Risks §4):** branch deletes go through `destroyBranchByName` → `WithBranchDeletionLock` (clones-map check under `cloneMutex`) + live `zfs list -t filesystem` (Task 4); snapshots rely on non-force `zfs destroy`. The sweeper never re-implements the clone check; it does not iterate the clones map itself, so it cannot propagate `destroyIdleClones`'s race
- [x] **bound the blast radius (Risks §3):** `deletionBudget` caps deletions per sweep at `maxDeletionsPerTick`; a single summary `log()` line fires when the cap is reached
- [x] **coalesce event emission (Risks §3/§14):** the sweeper emits NO per-entity webhook/telemetry events — the Task-4 destroy helpers don't emit (the HTTP handlers do), so a bulk sweep produces only `log()` lines. Coalesced summary telemetry is deferred to Task 7
- [x] single-`error` returns throughout; ctx checked via `ctx.Err()`/`<-ctx.Done()` without a `select`-on-one-error; whole loop gated on `retentionEnabled()` (both windows 0 = disabled)
- [x] emits a `log.Msg` line on every auto-delete (snapshot and branch) — no silent deletion
- [x] table-driven tests: `nextDeleteState` (protected/deps-reset/schedule/expired/exact-now), `snapshotIsLeaf` (child/root/branch/clones/head each block), `branchHasDependents`, `branchHeadSet`, `skipAutoDelete` (automatic/`_pre`/no-sep/`_pre`-in-branch-name), `deleteAtChanged`, `parseDeleteAt`, `deletionBudget` (cap honored), retention config helpers
- [x] `make test` + `make run-lint` (0 issues) + `make build` green

### Task 7: Telemetry & delete notifications (mirror clones)

**Files:**
- Modify: `engine/internal/telemetry/telemetry.go` (event-name constants)
- Modify: `engine/internal/telemetry/events.go` (payload structs)
- Modify: `engine/internal/srv/branch.go`, `engine/internal/srv/routes.go`, `engine/internal/srv/auto_delete.go` (emit)
- Modify: `engine/internal/webhooks/events.go` (optional: branch/snapshot delete + protection-expiring events)
- Modify: telemetry `*_test.go`

- [x] added telemetry events + payloads: `snapshot_updated` (`SnapshotUpdated{ID, Protected}`), `snapshot_destroyed` (`SnapshotDestroyed{ID}`), `branch_updated` (`BranchUpdated{Name, Protected}`). **Naming reconciliation:** the plan said `snapshot_deleted`/`branch_deleted`, but the existing convention is `clone_destroyed`/`branch_destroyed` — so snapshot deletion is `snapshot_destroyed` (mirrors `CloneDestroyed`) and **branch deletion REUSES the pre-existing `branch_destroyed`** (already emitted by `deleteBranch`), not a new `branch_deleted`
- [x] emit on PATCH (`patchSnapshot`→`snapshot_updated`, `patchBranch`→`branch_updated`), manual delete (`deleteSnapshot` now emits `snapshot_destroyed` + its existing webhook; `deleteBranch` already emitted `branch_destroyed`), and auto-delete — the sweeper emits **asynchronously**: one goroutine per sweep (`emitAutoDeleteEvents`, never one per entity), collecting deleted IDs during the sweep; webhook sends go through a **non-blocking** `emitWebhook` (select+default drop+log) so the buffer-1 channel can't stall, telemetry POSTs are bounded by `maxDeletionsPerTick`
- [x] webhook deletion events: reused the existing `SnapshotDeleteEvent`/`BranchDeleteEvent` (no new types); the sweeper emits them non-blocking. Manual paths keep their existing blocking sends (human-paced, guaranteed delivery)
- [x] **DEFERRED "protection-expiring" warning webhooks** — ZFS-anchored entities have no per-entity "warning already sent" flag, so a naive impl re-fires every tick. Noted as v1 follow-up; not built
- [x] tests: `emitWebhook` (delivers with space, drops-without-blocking when full), `mapKeys`, and telemetry payload JSON-tag marshaling (`internal/telemetry/events_test.go`, a new file as the package had none)
- [x] `make test` (48 pkg ok) + `make run-lint` (0 issues) + `make build` + `-race` green

### Task 8: CLI commands for protection (branch & snapshot)

**Files:**
- Modify: `engine/cmd/cli/commands/branch/command_list.go`, `actions.go`
- Modify: `engine/cmd/cli/commands/snapshot/command_list.go`, `actions.go`
- Modify/Create: `engine/cmd/cli/commands/clone/actions.go` (extract shared `parseProtectedFlag`/`parseDurationMinutes` into a reusable spot, e.g. a small shared package) + tests
- Modify: CLI `*_test.go`

- [x] added `--protected` (grammar: `true`/minutes/`30m`/`2h`/`7d`/`0`=forever/`false`=off). **snapshot** got a clean `snapshot update SNAPSHOT_ID` subcommand (snapshot uses `Subcommands`). **branch deviation:** the `branch` command is flag-driven (it already multiplexes list/create/delete by args+flags, and `switch`/`commit`/`log` are sibling top-level commands), so `--protected` was added to it: `dblab branch --protected=<v> BRANCH_NAME` updates protection (dispatched in `list()` when a name + `--protected` are both present). This preserves the existing branch command structure rather than restructuring it into a subcommand tree (CLAUDE.md: "do not redesign fundamental parts of the architecture"). Both call the Task-5 client methods (`UpdateSnapshot`/`UpdateBranch`)
- [x] extracted the parser to `commands.ParseProtectedFlag`/`ParseDurationMinutes` (new `cmd/cli/commands/protection.go`). Made it **three-state** (`*bool`: nil = flag unset = no change) for the snapshot/branch `Protected *bool` requests; clone keeps its two-state `bool` via a local `protectedFlag` wrapper that derefs (nil→false), preserving clone behavior. Parser tests moved to `commands/protection_test.go`; `clone/actions_test.go` removed (its only content was those tests)
- [x] `Protected`/`DeleteAt` surfaced: **snapshot list** already emits them via the `SnapshotView`→`*Snapshot` JSON (Task-1 model fields, `omitempty`), no CLI change needed. **branch list** now fetches `ListBranchesView` (new client method; `ListBranches` refactored to delegate) and annotates each line: `[protected]` / `[protected until <RFC3339>]` / `[auto-delete at <RFC3339>]`; branch name stays the first token so existing name-grepping is unaffected
- [x] tests: `commands/protection_test.go` (three-state parsing incl. not-set→nil, true/false/durations, overflow), `branch/actions_test.go` (annotation, dedup-by-name, sorted formatted list)
- [x] `make test` (49 pkg ok) + `make run-lint` (0 issues) + `make build` green

### Task 9: Config examples, swagger spec, and README

**Files:**
- Modify: all `engine/configs/config.example.*.yml` (+ `config.yml`)
- Modify: `engine/api/swagger-spec/dblab_openapi.yaml`
- Modify: `README.md` and any branching docs

- [x] added a commented `retention:` block (all fields `0`/default, auto-deletion off) after `server:` in the 5 standard examples (`logical_generic`, `logical_rds_iam`, `physical_generic`, `physical_pgbackrest`, `physical_walg`). **Skipped `config.example.ci_checker.yml` and `config.example.rds_refresh.yaml`** (different config schemas, no `server:`/branching) and `config.yml` (gitignored local working config, not a tracked example)
- [x] swagger: added `PATCH /snapshot/{id}` (`updateSnapshot`) and `PATCH /branch/{branchName}` (`updateBranch`); new request schemas `UpdateSnapshot`/`UpdateBranch` (three-state `protected`, `protectionDurationMinutes`, `deleteAt`, mutual-exclusivity documented); added `protected`/`protectedTill`/`deleteAt` to the `Snapshot` and `Branch` response schemas
- [x] README: extended the feature list — time-based auto-deletion of unused branches/snapshots (safe-only) under "Recovery & management", and broadened the clone "Deletion protection" bullet to cover clones, branches, and snapshots. Detailed CLI usage lives in the external postgres.ai/docs branching how-tos this README links to
- [x] validated: all edited YAML parses (`yaml.safe_load`), `go test ./pkg/config/... ./configs/...` green, `make run-lint` (0 issues), `make build` green

### Task 10: Verify acceptance criteria
- [x] **protected entities cannot be deleted (manual / count-based retention / sweeper).** Enforced and unit-tested at each path: manual delete via `ensureNotProtected`→`ProtectedTillActive` (`TestProtectedTillActive` incl. past/forever/malformed); count-based via `getProtectedSnapshots` appended to `busySnapshots` (`TestGetProtectedSnapshots`, `TestExcludeBusySnapshots`); sweeper via `nextDeleteState` (protected → no schedule) + pre-destroy `isProtectedNow` re-read (`TestNextDeleteState`). Full live-handler 4xx is a Post-Completion manual check (no httptest harness in `srv`)
- [x] **unused unprotected entities auto-deleted after retention; clock resets on a new dependent.** `TestNextDeleteState` covers first-seen→schedule, not-reached→keep, reached→delete, has-deps→clear, protected→clear; `TestSnapshotIsLeaf`/`TestBranchHasDependents` cover the dependency predicate
- [x] **protection and scheduled deletion never coexist.** `TestApplyProtectionUpdate` (protect clears delete_at, deleteAt clears protection, both-set rejected, empty rejected); the sweeper's `nextDeleteState` only schedules on unprotected entities
- [~] **multi-pool: properties resolve to the correct pool/dataset.** By construction: snapshots are pool-scoped (`detectPoolName`/`GetFSManager`); branch PATCH fans out across pools via `branchDatasets`; the sweeper reads each pool's own local prop. Verified by design + code review; a live multi-pool run remains a Post-Completion manual check
- [x] full suite: `make test` (49 pkg, 0 failures), `make run-lint` (0 issues), `make build` (binaries built) — all green
- [~] integration tests: `make test-ci-integration` (`go test -race -tags=integration ./...`) is **not feasible in this environment** (the `integration`-tagged tests under `diagnostic`/`retrieval`/`embeddedui` need Docker images the maintainer provisions, and none cover branching/protection). Verified the integration build still compiles with `go vet -tags=integration ./...` (exit 0). Deferred to CI

**Live e2e verification (against a running ZFS-backed instance, 2026-06-19):** seeded a golden copy + dbmarker, enabled `retention` (1-min cadence), and drove the real API/CLI:
- 🐛 **Bug found & fixed live:** `ListProtection` built `zfs get … -r <pool> <props>` (property list and dataset swapped) → zfs `invalid property`; broke `createSnapshot`/`CleanupSnapshots`/the sweeper. Fixed arg order; hardened the two masking unit tests to assert the exact command (commit `c1550fac`).
- ✅ snapshot PATCH: protect→forever, **manual delete refused 400 "is protected"**, protect+deleteAt rejected, deleteAt clears protection, unprotect leaves deleteAt
- ✅ branch PATCH timed protection, **manual delete refused 400**
- ✅ CLI: branch-update flag, snapshot-update subcommand, `BRANCH_NAME is required` guard, `dblab branch` shows `[protected until …]`
- ✅ **inheritance (Risks §2):** clone in a protected branch is `protected:false`; a commit snapshot under a protected branch dataset shows its own `protected=false` (not the inherited value)
- ✅ **sweeper:** unused branches `sweepme` and `b1` auto-deleted (log `auto-deletion: deleted unused branch …`); a branch with a clone and a protected branch both survived; `b1` was deleted only after its clone was destroyed (dependency clock)
- ✅ **lineage (Risks §1):** the intermediate commit `branch/lin@…` (has a child) survived the sweeps — non-leaf snapshots are never deleted
- multi-pool still unverified live (single-pool instance) — remains the only Post-Completion gap

### Task 11: [Final] Documentation & cleanup
- [x] README/swagger reflect final behavior (Task 9, reviewed in `03bc5bc4`; `protectionDurationMinutes`/retention semantics corrected post-review)
- [x] `CLAUDE.md` left unchanged — no genuinely new pattern emerged. The one live lesson (command-builder tests must assert the exact command string, not return canned output for any input — what hid the `ListProtection` bug) is captured in the Task 10 notes and the hardened `branching_test.go`, not worth a new global rule
- [x] moved this plan to `docs/plans/completed/`

## Outcome
Tasks 1–11 complete. Engine + API client + CLI + config + swagger + README done; unit-tested, linted, built; live e2e verified against a running ZFS instance (found & fixed the `ListProtection` arg-order bug, `c1550fac`). Only remaining gap: live **multi-pool** verification (the test instance is single-pool) — listed under Post-Completion. UI remains a separate follow-up (out of scope).

## Risks & intersections

Found by adversarial investigation of the codebase. Each is addressed in the tasks above; numbered here for reference.

1. **`HasDependentEntity` return-value trap (CRITICAL).** `branching.go:703-729` checks `dle:root` and `dle:child`
   but only **logs** them — it returns *only the dependent clones*, and `strings.Split("", ",")` yields `[""]`
   (length 1) when there are none. A sweeper that uses its return value to mean "no dependents" would delete
   intermediate branch-history snapshots (commit nodes with `dle:child` but no clones), **breaking branch lineage**
   (`dblab log`, parent traversal). → Task 6 uses an explicit true-leaf predicate and ignores this function's return.
2. **ZFS property inheritance (CRITICAL).** Clones live at `pool/branch/<name>/<clone>/r<n>` (nested under the branch
   dataset) and snapshots inherit dataset props; **no read uses `-s local`**. A branch-level `dle:protected_till` /
   `dle:delete_at` on `pool/branch/<name>` is therefore inherited by every clone/snapshot beneath it, and reads can't
   tell local from inherited. → Tasks 2/4/6 use the `-s local` readers (`GetProtection`/`ListProtection`) for all
   *authoritative* decisions. Review hardening also routes the **snapshot-list display** through `ListProtection`, so
   `dblab snapshot list` shows each snapshot's own protection, not a value inherited from a protected branch dataset.
   The `repoFields`/`getRepo` path remains inheritance-aware and is used only for branch repo/log display.
3. **Webhook/telemetry flood + per-tick blast radius.** Webhook channel is buffer-1 (main.go:116); telemetry
   `SendEvent` is a synchronous POST (telemetry.go:65). A first-tick-after-downtime sweep over many expired entities
   would stall/flood. → Task 6 caps `maxDeletionsPerTick` and **coalesces emission per sweep** (refined in §14 —
   per-entity async goroutines still pile up); Task 7 defers "expiring" warnings.
4. **Concurrency with the idle-checker and refresh.** → Task 6 copies the `checkProtectionLeases` locking pattern
   (not the racy `destroyIdleClones`), respects `cloneMutex → snapshotMutex → zfs.mu`, and sweeps only pools whose
   locked `pool.Status()` is `ActivePool` (skips `RefreshingPool` AND `EmptyPool`). The TOCTOU backstop differs by
   entity: snapshots rely on **non-force `zfs destroy`** (refuses if a clone appeared); **branches have NO such
   backstop** — see §8 and "Corrected branch-deletion design".
5. **`CleanupSnapshots` retention cron bypasses handler checks.** Raw `zfs destroy -R` shell pipeline. → Task 4 adds
   protected names to its `busySnapshots` exclude set (the only available lever).
6. **`sessions.json` persisted clones.** A swept snapshot referenced by a persisted clone → dangling on restart.
   Safe by construction (zero-clone requirement + non-force destroy refuses). Verified: `RestoreClonesState` calls
   `IncrementCloneNumber` for each restored clone (storage.go), so the count reflects *restored* clones too — but it
   remains in-memory and can still drift (§9), so authoritative destroys use the live ZFS signals, not this count.
7. **`Move`/reset drop custom props.** `zfs send|recv` (`Move`) and dataset-recreate paths don't carry `dle:*`; a
   protected branch could silently lose protection after a `Move`. Pre-existing limitation (affects existing props
   too) — documented, low frequency.
8. **Branch destroy is `zfs destroy -R` with no ZFS backstop (CRITICAL, verified).** `destroyBranchDataset` →
   `DestroyDataset` issues `zfs destroy -R <pool>/branch/<name>` unconditionally (zfs.go:503). DLE clones are nested
   *inside* the branch dataset (`<pool>/branch/<name>/<clone>/r<n>` — `branching.CloneName`), so they are descendants
   and BOTH `-r` and `-R` cascade-destroy them; switching `-R`→`-r` does NOT help. The only guard is the in-memory
   `GetCloneNumber` pre-check (branch.go:580), and `CreateClone` registers the clone in memory (under `cloneMutex`)
   *before* a detached goroutine runs `zfs clone` with no lock held (base.go:210 → goroutine → zfs.go:215). So the
   manual delete path is already racy and the unattended sweeper makes it material. → "Corrected branch-deletion
   design": atomic check-and-destroy under the cloning lock + a live `zfs list -t filesystem` descendant check.
9. **`GetCloneNumber` is in-memory, not live ZFS (verified).** It returns `snapshot.NumClones` from the `SnapshotBox`
   map (snapshots.go:170); it reflects restored clones but can drift from ZFS truth. → Destroys use authoritative live
   signals: snapshots use the native ZFS `clones` property (already read into `SnapshotProperties.Clones`,
   routes.go:317) + non-force destroy; branches use the live `zfs list -t filesystem` descendant check (§8). In-memory
   counts are an advisory pre-filter only.
10. **PATCH vs sweeper is not serialized on ZFS-property writes (verified).** `setProperty` takes no lock; `Manager.mu`
    guards only the snapshots slice. Mutual exclusivity is therefore eventually consistent, not atomic. → The sweeper
    **re-reads `GetProtection(target)` immediately before destroy** and aborts if now-protected; the design is
    self-healing (see "Protection ↔ deletion atomicity").
11. **Multi-pool branch protection gap (verified).** A branch dataset is per-pool, but `getFSManagerForBranch`
    resolves a branch via `s.pm.First()` only (branch.go:127), while the sweeper iterates all pools. A same-named
    branch could be protected on one pool and swept on another. → PATCH fans protection out to the branch dataset on
    every pool that has it; the sweeper reads each pool's own local prop (see "Multi-pool branch semantics").
12. **Sweeper must skip every non-`Active` pool (verified).** Pool status is `active`/`refreshing`/`empty`
    (resources/pool.go); it flips to `ActivePool` only after `RefreshData`+`SnapshotData` complete (retrieval.go), so a
    non-Active pool may be mid-refresh or failed. → Sweep only `pool.Status() == ActivePool`, re-checked per pool.
13. **Per-entity `zfs get` does not scale (verified).** A tick over thousands of snapshots × 2 props would spawn
    thousands of subprocesses. → **Resolved:** `ListProtection` batches one `zfs get -s local -t snapshot -r <pool>`
    per pool; `GetProtection` batches both props in one call for the single-entity pre-destroy re-read. The cloning
    snapshot-list also caches the result (refresh on reload), so reads do not run a ZFS call per request.
14. **Async event emit still piles up (verified).** Webhook channel is buffer-1 (main.go:116) and telemetry POST is
    synchronous (telemetry.go:65); a burst of per-entity emit goroutines blocks on the channel/POST. → Coalesce to one
    summary event per sweep, or non-blocking send with `default` drop+log.

## Technical Details

### ZFS property semantics
- `dle:protected_till`: empty (`-`) → not protected; `forever` → indefinite (`Protected=true`, `ProtectedTill=nil`);
  UTC RFC3339 → protected until then (`Protected=true`, `ProtectedTill=ts`). Stored on the snapshot, or on the branch
  dataset (`<pool>/branch/<name>`) for branches.
- `dle:delete_at`: empty (`-`) → no scheduled deletion; UTC RFC3339 → scheduled. Only ever set on unprotected entities.
- **Set/clear convention:** `setProperty(prop, value, target)` writes the value (mapping `""`→`"-"`); clearing =
  `setProperty(prop, "", target)`. `getProperty` returns `strings.Trim(strings.TrimSpace(out), "-")`. Timestamps are UTC RFC3339 with
  `Z`, which never starts/ends with `-`, so the trim is lossless. **Do not** use `zfs inherit` (inconsistent with the
  codebase's set-to-`-` convention).
- **Branch repo/log display reads** go through `repoFields` (tab-delimited, robust), never the positional
  `defaultFields`/`listDetails` parser (whitespace-split, fixed indices — see Task 3 warning). **Authoritative**
  protection/delete-at reads (delete and auto-delete decisions) and the **snapshot-list display** use the `-s local`
  readers `GetProtection`/`ListProtection` to ignore inherited values (Risks §2).

### "Unused" / true-leaf definition (Risks §1)
- **Snapshot deletable** = a true leaf: `dle:child` empty AND `dle:root` empty AND `dle:branch` empty (not a branch
  head) AND zero clones (`GetCloneNumber`) AND not in `getBranchHeadSnapshots()`. Do **not** rely on
  `HasDependentEntity`'s return value — it returns only clones (and `[""]` for none), not child snapshots/branches.
- **Branch deletable** = none of its snapshots have clones **and** no child branch forks from it (reuse the
  dependency walk already in `deleteBranch`).

### Sweeper decision logic (per entity, each tick)
1. `Protected` → ensure `delete_at` cleared; skip.
2. has dependents → clear `delete_at` (reset clock); skip.
3. no dependents and `delete_at` unset → set `delete_at = now + retention`.
4. no dependents and `now >= delete_at` → delete via `destroySnapshotByID` / `destroyBranchByName`.

This is self-healing across restarts and never deletes earlier than `retention` after the entity is observed unused.

### New config (`retention` section — exact wiring confirmed during Task 6)
```yaml
retention:
  unusedSnapshotMinutes: 0   # auto-delete unused snapshots after N minutes of no dependents; 0 = disabled
  unusedBranchMinutes: 0     # auto-delete unused branches after N minutes of no dependents; 0 = disabled
  checkIntervalMinutes: 5    # sweep cadence
  protectionMaxDurationMinutes: 0  # cap for timed protection; 0 = no cap
  maxDeletionsPerTick: 50    # blast-radius cap per sweep (Risks §3); excess deferred to the next tick
```
(Field names/section placement finalized in Task 6 against the existing config-parsing structure; mirror the
`cloning.maxIdleMinutes` precedent.)

### Mutual exclusivity (enforced in PATCH handlers)
- `protected=true` → clear `delete_at`.
- explicit `deleteAt` set → clear protection.
- the sweeper only writes `delete_at` on entities where protection is absent.

### Corrected branch-deletion design (Risks §8/§9)
**Problem.** `destroyBranchDataset` → `DestroyDataset` issues `zfs destroy -R <pool>/branch/<name>` unconditionally
(zfs.go:503). DLE clones are nested *inside* the branch dataset (`<pool>/branch/<name>/<clone>/r<n>` —
`branching.CloneName`), so they are descendants and `-R` (or even `-r`) cascade-destroys them. Unlike the snapshot
path, there is **no ZFS-level refusal** for branches. The only guard is the in-memory `GetCloneNumber` pre-check
(branch.go:580), and `CreateClone` registers the clone in memory under `cloneMutex` **before** a detached goroutine
runs the unlocked `zfs clone` (base.go:210 → goroutine → zfs.go:215). The manual delete path is therefore already
racy; the unattended sweeper makes it material.

**Single safe entry point** used by BOTH the HTTP handler and the sweeper:

1. `destroyBranchByName(branchName string) error` (srv): refuse a protected branch (`ensureNotProtected`), resolve fsm +
   the branch's snapshots (existing `snapshotsToRemove`), then perform the check-and-destroy **atomically w.r.t. clone
   registration**, then run the post-steps (`cleanupSnapshotProperties`, `RefreshSnapshotList`) *after* releasing the
   lock; events are emitted by the handler.
2. Atomicity via a cloning-service method (it owns the clones map):
   ```go
   func (c *Base) WithBranchDeletionLock(snapshotIDs []string, destroy func() error) error
   ```
   Acquire `cloneMutex`; check the **clones map** (`cloneDependentSnapshotLocked`) for any registered clone of the
   listed snapshots; only if none, call `destroy()` while still holding `cloneMutex`. Because registration
   (`setWrapper`, under `cloneMutex`) precedes the `zfs clone`, a concurrent create is either (a) already in the map →
   refused, or (b) blocked on `cloneMutex` until after the destroy → its later `zfs clone -p` cannot resurrect a
   destroyed clone. **The map is checked, not `NumClones`** — `IncrementCloneNumber` runs under `snapshotMutex` at a
   slightly later point than registration, so a clone can be registered-but-uncounted; checking the map (the structure
   `cloneMutex` actually freezes) closes that window and means `snapshotMutex` is not needed here.
3. `destroy()` calls a new ZFS-manager method `DestroyBranchDataset(branchDataset string) error`:
   ```
   children := zfs list -H -r -t filesystem -o name <branchDataset>
   if any child != <branchDataset> { return error }   // catches out-of-band / restored clones
   zfs destroy -R <branchDataset>                      // only the branch fs + its snapshots remain
   ```
   The locked map check catches clones *registered but not yet provisioned* (which `zfs list` would miss); the live
   `zfs list` catches clones *provisioned but absent from the in-memory map* (drift/restore). Together they are
   complete. `-R` is retained but is only ever reached when no clone descendant exists.
4. Keep the lock hold minimal — an unused branch's `-R` destroys only metadata (its commit snapshots), so it is fast;
   branch GC is rare and capped per tick. The sweeper's branch destroy must NOT route through `DestroyClone` (which
   also takes `cloneMutex` → deadlock).
5. **What the lock does and does NOT serialize.** The held `cloneMutex` serializes clone *registration*
   (`setWrapper`), NOT the lock-free `zfs clone` that runs later in `CreateClone`'s detached goroutine. Completeness
   rests on the union: a clone that *registered* before the held section is in the clones map → refused; a clone
   *provisioned* with no in-memory wrapper (restore/drift/out-of-band) is caught by the live `zfs list`. A new
   `CreateClone` cannot register (hence cannot reach `zfs clone`) until the lock is released — so no fresh `zfs clone`
   starts mid-destroy. Residual is benign: a create blocked on the lock proceeds after the dataset is gone and its
   `zfs clone -p` errors / makes a fresh empty branch, never resurrecting a destroyed clone.

**Alternative considered & rejected — rename-then-destroy.** `zfs rename` the branch dataset to a tombstone so new
clones (which `zfs clone -p` would recreate at the original path) can't land in the doomed dataset, then destroy the
tombstone. Rejected: disturbs descendant mountpoints, leaves a phantom branch via `-p`, and is more ZFS gymnastics
than the lock-based approach, which reuses the existing clone-lifecycle mutex pattern.

### Sweeper safety model — authoritative vs advisory signals
- **Snapshot "has clones" (authoritative):** native ZFS `clones` property (`SnapshotProperties.Clones`, used at
  routes.go:317) + non-force `zfs destroy` (refuses on clones/holds). In-memory `GetCloneNumber`/`NumClones` is an
  advisory pre-filter only (Risks §9).
- **Snapshot lineage (the ONLY guard against breaking `dblab log`):** ZFS does not know about
  `dle:child`/`dle:root`/`dle:branch`; non-force `zfs destroy` will happily delete an intermediate commit snapshot.
  The true-leaf predicate (all of `dle:child`, `dle:root`, `dle:branch` empty + not in `getBranchHeadSnapshots()` +
  `Clones` property empty) is therefore load-bearing, not defense-in-depth — heaviest test coverage of any piece.
- **Branch "has clones":** the locked + live check in the branch-deletion design above; the `-R` has no ZFS backstop,
  so this check is the whole safety.
- **`HasDependentEntity` `[""]`:** never use its return as a count. `deleteSnapshot` is incidentally safe because
  `branching.ParseCloneName("")` fails and the entry is skipped (routes.go:279-284) — the extracted
  `destroySnapshotByID` MUST preserve that skip.

### Protection ↔ deletion atomicity (no shared ZFS-write lock — Risks §10)
`setProperty` takes no lock, so PATCH and the sweeper write `dle:*` independently; mutual exclusivity is **eventually
consistent**, not atomic, but self-healing:
- The sweeper **re-reads `getLocalProperty(dle:protected_till)` immediately before destroy** and aborts if
  now-protected — closing the "protect during this tick" window to a single get→destroy.
- PATCH-protect always clears `dle:delete_at`; the sweeper clears `dle:delete_at` on any protected entity it sees. A
  transient both-set state converges within one tick and never authorizes a delete (the pre-destroy re-read gates it).

### Multi-pool branch semantics (Risks §11)
A branch is logically global but its dataset is per-pool (`<pool>/branch/<name>`), and `getFSManagerForBranch`
resolves via `s.pm.First()` only (branch.go:127).
- **PATCH protect/unprotect fans out** to the branch dataset on *every* pool that has it (iterate `GetFSManagerList`,
  set the prop where `<pool>/branch/<name>` exists).
- **The sweeper reads each pool's own branch-dataset local prop** and treats per-pool datasets independently for the
  leaf check; a branch is fully deleted only when unused on all pools. Document this scope explicitly — never let a
  branch be protected on pool A yet swept on pool B.

## Post-Completion
*Items requiring manual intervention or external systems — informational only.*

**Manual verification:**
- On a live ZFS DBLab instance: protect a snapshot/branch, confirm delete is refused; unprotect and confirm it
  deletes; create a clone on a snapshot, let it idle to confirm the sweeper resets the clock while the clone exists,
  then remove the clone and confirm scheduled auto-deletion fires after the retention window.
- Verify behavior across **multiple pools**.
- Confirm count-based retention (data refresh) leaves protected snapshots intact.
- **Inheritance check (Risks §2):** protect a branch, create a clone in it, confirm the clone is NOT reported
  protected and that `GetProtection` returns empty on the clone; confirm the snapshot list shows the branch's commit
  snapshots as unprotected (each shows only its own local value) and the sweeper ignores inherited `delete_at`.
- **Lineage check (Risks §1):** commit several times on a branch, confirm the sweeper never deletes an intermediate
  (non-head) commit snapshot and that `dblab log` stays intact.

**External / follow-up:**
- **UI** (`ui/`): expose protection toggle and `deleteAt` for branches/snapshots, **and add list polling/refresh** —
  background auto-deletion currently produces phantom rows until manual reload (lists load once into MobX stores, no
  polling/websocket; Risks §UX). Disable the delete button on protected entities. Separate task — out of scope here.
- **Behavior change to document/changelog:** delete now returns a "is protected" 4xx — external automation that loops
  "delete all snapshots/branches" will see new, expected failures.
- Update any telemetry dashboards to include the new branch/snapshot events.
- Consider days-friendly duration strings (`7d`) in config if maintainers want parity with the CLI grammar.

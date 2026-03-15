# Quality Engineering Guide

## PostgresAI / Database Lab Engine

This document defines the quality engineering standards, processes, and workflows for all PostgresAI products including Database Lab Engine (SE/EE) and platform UI components.

---

## Table of Contents

- [Core Philosophy](#core-philosophy)
- [Quality Layers](#quality-layers)
- [Layer 1: Automated Foundation](#layer-1-automated-foundation)
- [Layer 2: AI-Augmented Quality](#layer-2-ai-augmented-quality)
- [Layer 3: Human Quality Decisions](#layer-3-human-quality-decisions)
- [Development Workflow](#development-workflow)
- [Weekly Quality Rhythm](#weekly-quality-rhythm)
- [PostgreSQL-Specific Quality Standards](#postgresql-specific-quality-standards)
- [Quality Metrics](#quality-metrics)
- [Trust-Critical Failure Modes](#trust-critical-failure-modes)

---

## Core Philosophy

**Quality as Code** -- quality engineering is embedded into the development workflow itself, with AI amplifying every contributor. No separate QA team; quality ownership stays with engineers.

Three layers:

1. **Automated quality gates** catch 80%+ of issues before any human sees them
2. **AI-assisted review and testing** handles exploratory and edge-case work
3. **Human judgment** reserved for architecture decisions, customer-facing scenarios, and risk assessment

---

## Quality Layers

### Layer 1: Automated Foundation

All automated checks run as CI/CD pipeline stages. Every PR must pass all gates before merge.

#### 1.1 Static Analysis

| Check | Tool | Scope | When |
|-------|------|-------|------|
| Go linting | golangci-lint | engine/ | every PR |
| Go formatting | gofmt/goimports | engine/ | every PR |
| TypeScript/ESLint | eslint | ui/ | every PR |
| Style linting | stylelint | ui/ | every PR |
| Spell checking | cspell | ui/ | every PR |
| Secret scanning | gitleaks | repo-wide | pre-commit + CI |
| Security scanning | CodeQL | repo-wide | scheduled |

#### 1.2 Test Suites

| Suite | Command | When | Coverage |
|-------|---------|------|----------|
| Go unit tests | `make test` | every PR | all packages |
| Go integration tests | `make test-ci-integration` | MR pipeline | Docker-dependent |
| Bash integration tests | `engine/test/*.sh` | MR (PG 17-18), main (PG 9.6-18) | end-to-end flows |
| UI unit tests | `pnpm test` | every PR | React components |
| UI e2e tests | `pnpm cy:run` | every PR | Cypress flows |
| API contract tests | Newman/Postman | every PR | API endpoints |

#### 1.3 PostgreSQL Version Matrix

Full matrix on main branch; reduced set on feature branches to optimize pipeline time.

| Version | Feature Branch | Main Branch |
|---------|---------------|-------------|
| 9.6 | - | yes |
| 10-16 | - | yes |
| 17 | yes | yes |
| 18 | yes | yes |

#### 1.4 Build Verification

- All binaries (server, CLI, CI checker, RDS refresh) must build on every PR
- Docker images must build successfully
- Cross-platform CLI builds (darwin/linux/freebsd/windows, amd64/arm64) verified on main

#### 1.5 Performance Regression Detection

Automated benchmarks on merge to main with statistical comparison against baseline:

- Thin clone creation time
- Snapshot creation/restoration time
- API response latencies (p50, p95, p99)
- Memory usage under concurrent clone load

Track in `quality/metrics/benchmarks/` with per-commit results.

### Layer 2: AI-Augmented Quality

#### 2.1 AI-Assisted PR Review

Use the system prompt in `quality/prompts/pr-review-system-prompt.md` for automated review. The reviewer checks:

- PostgreSQL-specific correctness (connection handling, transaction safety, lock awareness)
- Error handling completeness (every DB operation must handle errors)
- Resource lifecycle (connections opened must be closed, advisory locks released)
- SQL safety (no raw concatenation, parameterized queries only)
- Concurrency safety (proper mutex usage, no data races)
- Configuration validation (bounds checking, sensible defaults)

#### 2.2 AI-Assisted Test Generation

Use the prompt in `quality/prompts/test-generation-prompt.md` when implementing new features. The AI generates test cases covering:

- Normal operation path
- Empty/nil/zero-value inputs
- Boundary conditions
- Concurrent access scenarios
- PostgreSQL version-specific behavior
- Extension compatibility edge cases

Developer reviews, adjusts, and owns the generated tests.

#### 2.3 Spec-to-Test Pipeline

For features with written specs:

1. Write spec in markdown (feature description, expected behavior, edge cases)
2. Feed spec to AI to generate acceptance test skeletons
3. Developer reviews, fills in implementation-specific details
4. Tests become the executable spec -- spec and tests stay in sync

#### 2.4 Automated Issue Triage

When a bug is reported:

1. AI classifies severity (critical/high/medium/low)
2. Identifies likely affected components from stack traces and logs
3. Searches for related past issues
4. Drafts initial investigation path
5. Human picks up with context already assembled

### Layer 3: Human Quality Decisions

Reserve human attention for:

- **Architecture reviews** for features touching data safety (clone creation, snapshot management, WAL interaction)
- **Customer scenario testing** before releases -- walk through key workflows manually:
  - "Clone a 500GB database in under 60 seconds"
  - "Run SAMO analysis on a production-like workload"
  - "Recover from a failed snapshot mid-operation"
- **Risk classification** for autonomous features -- every action that modifies PostgreSQL configuration or data needs a human-defined risk level and corresponding safety gate
- **Security review** for any code handling authentication, authorization, or direct SQL execution

---

## Development Workflow

### For Every Feature

```
1. Spec written
   -> reviewed by at least one other engineer
   -> fed to AI for gap analysis ("what failure modes aren't addressed?")

2. Implementation + tests
   -> developer writes code
   -> AI generates test scaffolding from spec
   -> developer refines tests
   -> target: 80%+ code coverage for new code

3. PR opened
   -> CI runs fast suite (unit tests, lint, build)
   -> AI runs PR review (see prompts/pr-review-system-prompt.md)
   -> human reviewer focuses on design and PostgreSQL correctness

4. Merge to main
   -> nightly full matrix runs
   -> performance benchmarks compared to baseline

5. Release candidate
   -> AI produces release readiness report (see checklists/)
   -> human does scenario walkthrough
   -> decision made
```

### PR Review Standards

Every PR must have:

- [ ] All CI checks passing (tests, lint, build)
- [ ] AI review completed with no unresolved critical findings
- [ ] At least one human approval
- [ ] New/modified code has corresponding tests
- [ ] No regression in test coverage
- [ ] Breaking API changes documented

See `quality/checklists/pr-review-checklist.md` for the full checklist.

### Commit Standards

- Present tense, imperative mood ("add feature" not "added feature")
- First line under 72 characters
- Detailed description in body when warranted
- All commits signed
- Reference related issues

---

## Weekly Quality Rhythm

### Monday

- Review test failures from weekend/nightly runs
- Triage any new issues reported over the weekend
- Review quality metrics dashboard for trends

### Wednesday (mid-week check)

- Review open PRs for stale reviews
- Check for flaky test patterns in recent CI runs
- Address any performance regression alerts

### Friday

- Quality retrospective: what slipped through this week?
- Does a new test need to be added?
- Does a CI check need tightening?
- Update quality metrics tracking

---

## PostgreSQL-Specific Quality Standards

These standards are non-negotiable given that DBLab interacts with production PostgreSQL instances.

### SQL Safety

- **No raw SQL concatenation.** All user-provided values must use parameterized queries.
- Every SQL query the product generates must be tested against `EXPLAIN ANALYZE` output for:
  - No sequential scans on large tables
  - No unexpected lock escalation
  - Appropriate index usage

### Connection Management

- Every database connection must have a timeout configured
- Connections must be returned to pool after use (defer pattern)
- Graceful degradation under connection exhaustion
- Connection pool sizing must be configurable and documented

### Extension Compatibility

Maintain a first-class extension compatibility matrix in CI:

| Extension | Priority | Tested Versions |
|-----------|----------|----------------|
| pg_stat_statements | critical | all PG versions |
| pg_stat_kcache | high | PG 14+ |
| auto_explain | high | all PG versions |
| PostGIS | medium | PG 14+ |
| pg_partman | medium | PG 14+ |
| pgvector | medium | PG 14+ |

### WAL and Replication Safety

Any feature touching WAL or replication requires specific tests for:

- Replica lag behavior
- Failover scenarios
- WAL segment cleanup
- Archive command compatibility

### Transaction Safety

- Document expected transaction isolation level for every DB operation
- Test behavior under concurrent access
- Verify no long-running transactions that could cause table bloat

### Destructive Testing

Maintain a destructive testing harness that simulates:

- Disk full during clone/snapshot operations
- OOM conditions
- Network partition between DLE and PostgreSQL
- Kill/restart mid-clone
- Corrupt ZFS snapshot recovery

---

## Quality Metrics

Track these metrics continuously. See `quality/metrics/` for tracking templates.

### Code Quality

| Metric | Target | Measurement |
|--------|--------|-------------|
| Unit test coverage | >= 80% | `go test -cover` |
| Lint violations | 0 on main | golangci-lint |
| Cyclomatic complexity | < 30 per function | golangci-lint |

### Pipeline Health

| Metric | Target | Measurement |
|--------|--------|-------------|
| CI pass rate | >= 95% | pipeline analytics |
| Flaky test rate | < 2% | test result tracking |
| Pipeline duration (fast) | < 10 min | pipeline analytics |
| Pipeline duration (full) | < 45 min | pipeline analytics |

### Defect Tracking

| Metric | Target | Measurement |
|--------|--------|-------------|
| Mean time to detection | < 24 hours | issue timestamps |
| Escaped defects per release | < 3 | post-release tracking |
| Critical bug fix time | < 4 hours | issue resolution time |

### Performance

| Metric | Target | Measurement |
|--------|--------|-------------|
| Clone creation (100GB) | < 60s | benchmark suite |
| API response (p95) | < 200ms | benchmark suite |
| Snapshot creation | no regression > 5% | benchmark comparison |

---

## Trust-Critical Failure Modes

These are the top failure modes that would break customer trust. Each must have dedicated automated coverage with explicit test cases.

### 1. Data Loss During Clone

- **Risk**: thin clone corruption, snapshot staleness, ZFS pool failure
- **Coverage**: destructive test harness, snapshot integrity checks, clone verification tests
- **Gate**: block release if any clone integrity test fails

### 2. Incorrect Diagnostic Recommendation (SAMO)

- **Risk**: wrong index suggestion, incorrect bloat detection, false positive on lock contention
- **Coverage**: known-answer test suite against reference databases, cross-version validation
- **Gate**: all diagnostic outputs validated against expert-reviewed baselines

### 3. Silent Monitoring Failure

- **Risk**: metrics stop collecting without alerting, stale data presented as current
- **Coverage**: heartbeat tests for all monitoring components, staleness detection
- **Gate**: alerting on any monitoring gap > 5 minutes

### 4. Security Exposure

- **Risk**: authentication bypass, SQL injection, credential leakage in logs
- **Coverage**: CodeQL scanning, secret detection (gitleaks), parameterized query enforcement
- **Gate**: zero critical/high security findings

### 5. Performance Regression

- **Risk**: clone creation slowdown, API latency increase, memory leak
- **Coverage**: automated benchmark suite with statistical comparison
- **Gate**: no regression exceeding 10% from baseline on any key metric

---

## Getting Started Checklist

For the first month, prioritize:

- [ ] Set up AI-assisted PR review with PostgreSQL-specific system prompt
- [ ] Ensure all 5 trust-critical failure modes have dedicated test coverage
- [ ] Instrument quality metrics (test coverage, CI pass rate, benchmark trends)
- [ ] Run first weekly quality retrospective
- [ ] Validate extension compatibility matrix in CI
- [ ] Create destructive testing harness (start with disk-full and kill-mid-clone)
- [ ] Document performance baselines for clone creation and API latency

---

*This is a living document. Update it as quality standards evolve and new failure modes are discovered.*

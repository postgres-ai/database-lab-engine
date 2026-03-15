# Release Readiness Checklist

Complete this checklist before every release. All `[BLOCKING]` items must pass.

---

## Version: _______________
## Release Date: _______________
## Release Manager: _______________

---

## 1. Code Freeze Verification

- [ ] `[BLOCKING]` All planned PRs merged
- [ ] `[BLOCKING]` No open PRs with "release-blocker" label
- [ ] `[BLOCKING]` Main branch CI pipeline fully green (all stages)
- [ ] All deprecation warnings addressed or documented
- [ ] CHANGELOG updated with all user-facing changes

## 2. Test Coverage

- [ ] `[BLOCKING]` Full PostgreSQL version matrix passes (9.6-18)
- [ ] `[BLOCKING]` All integration tests pass
- [ ] `[BLOCKING]` All bash integration tests pass across PG matrix
- [ ] `[BLOCKING]` Unit test coverage >= 80% for modified packages
- [ ] Performance benchmarks show no regression > 5%
- [ ] API contract tests (Newman) pass against release candidate
- [ ] UI e2e tests (Cypress) pass

## 3. Trust-Critical Verification

- [ ] `[BLOCKING]` Clone integrity: create, use, destroy 100 clones with data verification
- [ ] `[BLOCKING]` Snapshot safety: snapshot create/restore cycle with data checksum validation
- [ ] `[BLOCKING]` No security vulnerabilities (CodeQL + gitleaks clean)
- [ ] `[BLOCKING]` Connection handling: verified under connection pool exhaustion
- [ ] Destructive test harness: disk-full, OOM, kill-mid-clone scenarios pass

## 4. Compatibility

- [ ] Extension compatibility matrix verified (pg_stat_statements, auto_explain, etc.)
- [ ] Docker image builds for all targets (server, CLI, CI checker, RDS refresh)
- [ ] Cross-platform CLI builds verified (darwin, linux, windows; amd64, arm64)
- [ ] ZFS 0.8 variant image builds and passes smoke tests

## 5. Human Scenario Walkthrough

One engineer walks through each scenario on a release candidate build:

- [ ] Fresh installation with default configuration
- [ ] Upgrade from previous version (config migration)
- [ ] Create database clone and run queries against it
- [ ] Snapshot lifecycle (create, list, use, delete)
- [ ] API authentication flow (token-based)
- [ ] CLI basic operations (status, clone create/destroy, snapshot list)
- [ ] UI: instance overview, clone management, configuration editing
- [ ] Error recovery: restart DLE mid-operation and verify state consistency

## 6. Documentation

- [ ] API documentation updated at dblab.readme.io
- [ ] Release notes written (user-facing language, not technical jargon)
- [ ] Breaking changes highlighted with migration instructions
- [ ] New configuration options documented with examples

## 7. Release Artifacts

- [ ] Docker images tagged with correct version
- [ ] Docker images pushed to both GitLab registry and Docker Hub
- [ ] CLI binaries uploaded to GCP bucket
- [ ] Git tag created and signed

## 8. Post-Release

- [ ] Verify Docker Hub images pull and run correctly
- [ ] Verify CLI binary download and basic operation
- [ ] Monitor error tracking for 24 hours post-release
- [ ] Update "latest" tags/links if stable release

---

## Sign-Off

| Role | Name | Date | Approved |
|------|------|------|----------|
| Release Manager | | | [ ] |
| Engineering Lead | | | [ ] |

---

## Notes

_Add any release-specific notes, known issues, or follow-up items here._

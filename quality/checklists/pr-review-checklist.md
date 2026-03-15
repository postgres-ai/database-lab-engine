# PR Review Checklist

Use this checklist for every pull request. Items marked with `[CRITICAL]` are blocking -- the PR cannot merge if they fail.

---

## Automated Gates (CI Must Pass)

- [ ] `[CRITICAL]` Unit tests pass (`make test`)
- [ ] `[CRITICAL]` Linter passes (`make run-lint`)
- [ ] `[CRITICAL]` Build succeeds (`make build`)
- [ ] `[CRITICAL]` No secrets detected (gitleaks)
- [ ] Integration tests pass (if applicable)
- [ ] UI lint/test pass (if UI changes)

## Code Quality

- [ ] New/modified code has corresponding tests
- [ ] Test coverage for new code >= 80%
- [ ] No increase in cyclomatic complexity beyond threshold (< 30)
- [ ] Functions under 60 lines where practical
- [ ] Lines under 130 characters
- [ ] No deep nesting (max 3 levels)
- [ ] No else-if chains (use switch or early returns)
- [ ] Imports follow standard ordering (stdlib, third-party, project)

## PostgreSQL Safety

- [ ] `[CRITICAL]` No raw SQL string concatenation (use parameterized queries)
- [ ] `[CRITICAL]` All database connections have timeouts configured
- [ ] `[CRITICAL]` All opened connections/transactions are properly closed (defer pattern)
- [ ] No assumptions about default transaction isolation level
- [ ] Advisory locks acquired are released in all code paths
- [ ] SQL queries tested for appropriate index usage
- [ ] No sequential scans on large tables in generated queries

## Error Handling

- [ ] `[CRITICAL]` All errors from database operations are checked
- [ ] Error messages are descriptive and actionable
- [ ] Error wrapping uses `fmt.Errorf("context: %w", err)` pattern
- [ ] No panics in library code (return errors instead)
- [ ] Resource cleanup on error paths (connections, files, locks)

## Concurrency

- [ ] Shared state protected by appropriate synchronization
- [ ] No data races (verified by `go test -race`)
- [ ] Context cancellation respected in long-running operations
- [ ] Goroutine lifecycle managed (no leaks)

## API Changes

- [ ] Breaking changes documented and versioned
- [ ] OpenAPI spec updated for endpoint changes
- [ ] Request/response validation added
- [ ] Error responses follow project standards

## Configuration

- [ ] New config options have sensible defaults
- [ ] Config values are validated on startup
- [ ] Config changes are documented

## Security

- [ ] `[CRITICAL]` No credentials or secrets in code or logs
- [ ] `[CRITICAL]` Authentication/authorization checks present for new endpoints
- [ ] Input validation at system boundaries
- [ ] Log messages do not leak sensitive data

## Documentation

- [ ] Exported functions/types have godoc comments
- [ ] Non-obvious behavior is documented in code comments
- [ ] README updated if user-facing behavior changes
- [ ] Comments are lowercase (except godoc for public symbols)

## UI Changes (if applicable)

- [ ] TypeScript strict mode -- no `any` types
- [ ] Components have appropriate error boundaries
- [ ] Responsive layout verified
- [ ] Accessibility basics (labels, alt text, keyboard nav)
- [ ] No console.log statements left in code

---

## Reviewer Notes

- Focus human review on **design and PostgreSQL correctness**
- AI review handles style, obvious bugs, and common patterns
- If unsure about PostgreSQL behavior across versions, request a version-matrix test
- For changes touching clone/snapshot operations, request destructive testing verification

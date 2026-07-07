# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture Overview
### Core Components
1. **Backend code** (`engine/`)
   1.1. **Entry Points** (`cmd/`)
2. **Frontend code** (`ui/`)

### Notable patterns
- **Source-detection logic** lives in `engine/internal/retrieval/probe/`. The `Propose` orchestrator returns a structured `ProposedConfig` consumed by both the `/admin/probe-source` HTTP handler and the `dblab local-install` CLI (`engine/cmd/cli/commands/localinstall/`). Extend probe rules there, not in `tools/db/pg.go`.
- **Config projection** (`engine/pkg/models/configuration.go` + `engine/pkg/util/projection/`) walks only top-level struct fields with single flat `proj:"..."` paths — no recursion into nested structs. The `RetrievalMode` field on `ConfigProjection` carries no `proj:` tag because it's a synthetic field injected by `projectedAdminConfig` after `StoreJSON`; the request-side dispatcher reads it directly from the incoming JSON map. A projection field for a key that is **not pre-seeded** in the config scaffold (e.g. `connectionString`) needs the `,createKey` tag — `projection.Set` silently skips a missing leaf without it.
- **Mode-aware config writes**: `applyProjectedAdminConfig` dispatches on the synthetic `retrievalMode`, and `guardModeFields` enforces a per-mode allow-list of populated fields so logical-only fields cannot leak into a physical config (and vice-versa).
- **Source connection-string passthrough**: when `logicalDump.options.source.connectionString` is set it wins over the discrete `connection.*` fields. `logical/connstring.go` (`withDatabase`, `sourcePgxConfig`) preserves every libpq option (sslmode, connect_timeout, …) end-to-end into `pg_dump` (`-d <conninfo>`) and the engine's own pgx connections to the **source** (`getDBList`, `dbSourceActivity`). The password is always injected separately — never embedded in the string. Restore paths target the local container and must never receive it.
- **Engine-side image resolution**: `probe/registry.go` resolves a glibc-aware docker image by querying the live registries (Docker Hub for the generic image, GitLab for managed-provider SE images) with a per-repo TTL cache and a `go:embed` offline snapshot (`images_fallback.json`). Resolution never hard-fails or hangs: fresh cache → live fetch (bounded) → last good cache → embedded snapshot → provider default `<repo>:<major>`. Tag selection (`imageselect.go`) is pure and unit-tested; the `Registry` is built once on `Server`.

## Testing Before Pushing

**Always run tests before pushing.** If tests fail, do not push. Fix the failing tests first. Only skip this if the user explicitly says to ignore test failures.

## Build/Test/Lint Commands
- Build all components: `cd engine && make build`
- Lint code: `cd engine && make run-lint`
- Run unit tests: `cd engine && make test`
- Run integration tests: `cd engine && make test-ci-integration`
- Run a specific test: `cd engine && GO111MODULE=on go test -v ./path/to/package -run TestName`
- Run UI: `cd ui && pnpm start:ce` (Community Edition) or `pnpm start:platform`

## Code Style Guidelines
- Go code follows "Effective Go" and "Go Code Review Comments" guidelines
- Use present tense and imperative mood in commit messages
- Limit first commit line to 72 characters
- All Git commits must be signed
- Format Go code with `cd engine && make fmt`
- Use error handling with pkg/errors
- Follow standard Go import ordering
- Group similar functions together
- Error messages should be descriptive and actionable
- UI uses pnpm for package management

## Important Backend Workflow Notes

- Always run tests, linter and normalize comments BEFORE committing anything
- Run formatting, code generation, linting and testing on completion
- Never commit without running completion sequence
- Run tests and linter after making significant changes to verify functionality
- IMPORTANT: Never put into commit message any mention of Claude or Claude Code
- IMPORTANT: Never include "Test plan" sections in PR descriptions
- Do not add comments that describe changes, progress, or historical modifications
- Comments should only describe the current state and purpose of the code, not its history or evolution
- After important functionality added, update README.md accordingly
- When merging master changes to an active branch, make sure both branches are pulled and up to date first
- Don't leave commented out code in place
- Avoid multi-level nesting
- Avoid multi-level ifs, never use else if
- Never use goto
- Avoid else branches if possible
- Write tests in compact form by fitting struct fields to a single line (up to 130 characters)
- Before any significant refactoring, ensure all tests pass and consider creating a new branch
- When refactoring, editing, or fixing failed tests:
    - Do not redesign fundamental parts of the code architecture
    - If unable to fix an issue with the current approach, report the problem and ask for guidance
    - Focus on minimal changes to address the specific issue at hand
    - Preserve the existing patterns and conventions of the codebase

## Backend Code Style Guidelines

### Import Organization
- Organize imports in the following order:
    1. Standard library packages first (e.g., "fmt", "context")
    2. A blank line separator
    3. Third-party packages
    4. A blank line separator
    5. Project imports (e.g., "gitlab.com/postgres-ai/database-lab/v3/pkg/*")
- Example:
  ```go
  import (
      "context"
      "fmt"
      "net/http"

      "github.com/docker/docker/api/types"
      "github.com/gorilla/mux"

      "gitlab.com/postgres-ai/database-lab/v3/pkg/util/branching"
  )
  ```

### Error Handling
- Return errors to the caller rather than using panics
- Use descriptive error messages that help with debugging
- Use error wrapping: `fmt.Errorf("failed to process request: %w", err)`
- Check errors immediately after function calls
- Return early when possible to avoid deep nesting

### Variable Naming
- Use descriptive camelCase names for variables and functions
- Good: `notFoundHandler`, `requestContext`, `userID`
- Bad: `not_found_handler`, `x`, `temp1`
- Be consistent with abbreviations (e.g., `httpClient` not `HTTPClient`)
- Local scope variables can be short (e.g., "lmt" instead of "orderLimit")
- Use constants for magic numbers and strings
- Use meaningful names for constants and enums

### Function Parameters
- Group related parameters together logically
- Use descriptive parameter names that indicate their purpose
- Consider using parameter structs for functions with many (4+) parameters
- If function returns 3 or more results, consider wrapping in Result/Response struct
- If function accepts 3 or more input parameters, consider wrapping in Request/Input struct (but never add context to struct)

### Documentation
- All exported functions, types, and methods must have clear godoc comments
- Begin comments with the name of the element being documented
- Include usage examples for complex functions
- Document any non-obvious behavior or edge cases
- All comments should be lowercase, except for godoc public functions and methods
- IMPORTANT: all comments except godoc comments must be lowercase, test messages must be lowercase, log messages must be lowercase

### Code Structure
- Keep code modular with focused responsibilities
- Limit file sizes to 300-500 lines when possible
- Group related functionality in the same package
- Use interfaces to define behavior and enable mocking for tests
- Keep code minimal and avoid unnecessary complexity
- Don't keep old functions for imaginary compatibility
- Interfaces should be defined on the consumer side (idiomatic Go)
- Aim to pass interfaces but return concrete types when possible
- Consider nested functions when they simplify complex functions

### Code Layout
- Keep cyclomatic complexity under 30
- Function size preferences:
    - Aim for functions around 50-60 lines when possible
    - Don't break down functions too small as it can reduce readability
    - Maintain focus on a single responsibility per function
- Keep lines under 130 characters when possible
- Avoid if-else chains and nested conditionals:
    - Never use long if-else-if chains; use switch statements instead
    - Prefer early returns to reduce nesting depth
    - Extract complex conditions into separate boolean functions or variables
    - Use context structs or functional options instead of multiple boolean flags

### Testing
- Write thorough tests with descriptive names (e.g., `TestRouter_HandlesMiddlewareCorrectly`)
- Prefer subtests or table-based tests, using Testify
- Use table-driven tests for testing multiple cases with the same logic
- Test both success and error scenarios
- Mock external dependencies to ensure unit tests are isolated and fast
- Aim for at least 80% code coverage
- Keep tests compact but readable
- If test has too many subtests, consider splitting it to multiple tests
- Never disable tests without a good reason and approval
- Important: Never update code with special conditions to just pass tests
- Don't create new test files if one already exists matching the source file name
- Add new tests to existing test files following the same naming and structuring conventions
- Don't add comments before subtests, t.Run("description") already communicates what test case is doing
- Never use godoc-style comments for test functions

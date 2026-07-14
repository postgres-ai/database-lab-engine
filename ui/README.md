# DBLab Engine UI and DBLab Platform UI

## DBLab - thin database clones and database branching for faster development

_See the [DBLab Engine repository](https://gitlab.com/postgres-ai/database-lab) for more information about the underlying technology._
DBLab Engine is an open-source (Apache 2.0) solution that clones Postgres databases of any size in seconds. This capability helps solve common challenges, such as:

- Build dev/QA/staging environments with full-size, production-like databases.
- Provide temporary full-size database clones for SQL query analysis and optimization.
- Automatically verify database migrations (schema changes) and large data operations in CI/CD pipelines to minimize the risk of downtime and performance degradation.

For example, cloning a 10 TiB Postgres database can take less than 2 seconds.

## Development

### List of packages:

- `@postgres.ai/ce` - Community Edition UI package
- `@postgres.ai/shared` - Shared modules and utilities

## UI development documentation

At the repository root, you can run commands for all packages or individual packages:

- `pnpm --filter <package-name> <command>` – run the specified command on a single package.

#### Examples
- `pnpm install` – install all dependencies.
- `pnpm --filter @postgres.ai/ce build` – build the Community Edition UI.
- `pnpm --filter @postgres.ai/ce start` – run the Community Edition UI locally in development mode.

_Important note: do not run or build the `@postgres.ai/shared` package directly; it is a dependency._

### How to start the Community Edition UI
- `cd ui`
- `pnpm install` – install dependencies for all packages (run once).
- `pnpm --filter @postgres.ai/ce start` – start the development server.

The dev server proxies `/api` and `/ws` to `http://localhost:446` by default.
Set the `VITE_DEV_PROXY_TARGET` environment variable to override the proxy target, for example:
`VITE_DEV_PROXY_TARGET=https://demo.dblab.dev:446 pnpm --filter @postgres.ai/ce start`

### How to build the Community Edition UI

- `cd ui`
- `pnpm install` – install dependencies for all packages (run once).
- `pnpm --filter @postgres.ai/ce build` – build the Community Edition UI.

### CI pipelines for UI code

To deploy UI changes, tag the commit with a `ui/` prefix and push it. For example:

```shell
git tag ui/1.0.12
git push origin ui/1.0.12
```

## Vulnerability issues
Vulnerabilities, CVEs, and security issues can be reported on GitLab or GitHub through the tools and bots we use to ensure that DBLab Engine code remains safe and secure. Below we outline two primary categories: known CVEs in dependencies and issues detected by static analysis tools.

#### Package issues
Ways to resolve (in descending order of preference):
1. Update the package – search npm for a newer version, as the vulnerability may already be fixed.
2. If the vulnerability is in a sub-package, use [`pnpm.overrides`](https://pnpm.io/package_json#pnpmoverrides) in the root `package.json` to pin the transitive dependency to a patched version. Use this technique with caution — it may break the project during build or at runtime. Perform a full end-to-end test afterward.
3. Fork the package and include it locally in this repository.
4. If the issue is a false positive vulnerability, ignore it using your SAST tool's ignore directives. **This should be the last resort; apply other solutions first.**

#### Code issues
Ways to resolve (in descending order of preference):
1. If a portion of the source code is written in `.js`, rewrite it in `.ts` or `.tsx` — this can resolve many potential security issues.
2. Follow your SAST tool's recommendations and apply fixes manually or automatically.
3. If the finding is a false positive, ignore it using your SAST tool's ignore directives. **This should be the last resort; apply other solutions first.**

<!-- TODO: move this ^ to the main README.md and CONTRIBUTING.md -->

## Migrating to TypeScript
- `@postgres.ai/shared` is written in TypeScript.
- `@postgres.ai/ce` is written in TypeScript.
- There may be typing issues: older packages might lack type definitions. It is recommended to update or replace them. If that is not possible, write a custom definition file named `<package-name>.d.ts` in the `src` directory of the appropriate package.

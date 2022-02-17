# Database Lab Engine and Database Lab Engine UI

## Database Lab - thin database clones for faster development
_Proceed to [Database Lab Engine repository](https://gitlab.com/postgres-ai/database-lab) for more information about technology itself._
Database Lab Engine (DLE) is an open-source (AGPLv3) technology that allows blazing-fast cloning of Postgres databases of any size in seconds. This helps solve many problems such as:
- build dev/QA/staging environments involving full-size production-like databases,
- provide temporary full-size database clones for SQL query analysis optimization,
- automatically verify database migrations (DB schema changes) and massive data operations in CI/CD pipelines to minimize risks of downtime and performance degradation.

As an example, cloning a 10 TiB PostgreSQL database can take less than 2 seconds.

## Development
### List packages:
- `@postgres.ai/platform` - platform version of UI
- `@postgres.ai/ce` - community edition version of UI
- `@postgres.ai/shared` - common modules

### How to operate
At the root:
- `<npm command> -ws` - for all packages
- `<npm command> -w <package-name>` - for specific package

#### Examples
- `npm ci -ws` - install deps of all packages
- `npm build -ws` - build all packages
- `npm start -w @postgres.ai/platform` - run platform UI locally in dev mode
- `npm start -w @postgres.ai/ce` - run community edition UI locally in dev mode

_Important note: don't use commands for `@postgres.ai/shared` - it's dependent package, which can't be running or built_

### How to start
- `npm ci -ws`
- `npm start -w @postgres.ai/platform` or `npm start -w @postgres.ai/ce`

### How to build
- `npm ci -ws`
- `npm build -ws`

### CI pipelines for UI code
To deploy UI changes, tag the commit with `ui/` prefix and push it. For example:
```shell
git tag ui/1.0.12
git push origin ui/1.0.12
```

## Vulnerability issues

### Packages issues
Ways to resolve (ordered by preference):
1. Update a package - try to looking for a newer package in npm, probably this vulnerability are already fixed.
2. If vulnerability placed in a sub-package - try to replace it  with [npm-force-resolutions](https://www.npmjs.com/package/npm-force-resolutions). Be careful using this way - it may break a project as in a build phase as at runtime. Recommended full e2e testing after replacing.
3. Fork the package and put it locally in this repo.
4. If you are sure this is a falsy vulnerability - try to ignore it using special commands for your SAST tool. **NOT RECOMMENDED**.

### Code issues
Ways to resolve (ordered by preference):
1. If the part of source code is written on `.js` try to rewrite it on `.ts` or `.tsx` - it will fix a lot of potential security issues.
2. Follow the recommendations of your SAST tool - fix it manually or automatically.
3. If you are sure this is a falsy vulnerability - try to ignore it using special commands for your SAST tool. **NOT RECOMMENDED**.

<!-- TODO: move this ^ to the main README.md and CONTRIBUTING.md -->
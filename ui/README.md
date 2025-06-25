# Database Lab Engine UI and DBLab Platform UI

## DBLab - thin database clones and database branching for faster development

_Proceed to [Database Lab Engine repository](https://gitlab.com/postgres-ai/database-lab) for more information about technology itself._
Database Lab Engine (DLE) is an open-source (Apache 2.0) technology that allows blazing-fast cloning of Postgres databases of any size in seconds. This helps solve many problems such as:

- build dev/QA/staging environments involving full-size production-like databases,
- provide temporary full-size database clones for SQL query analysis optimization,
- automatically verify database migrations (DB schema changes) and massive data operations in CI/CD pipelines to minimize risks of downtime and performance degradation.

As an example, cloning a 10 TiB PostgreSQL database can take less than 2 seconds.

## Development

### List packages:

- `@postgres.ai/platform` - platform version of UI
- `@postgres.ai/ce` - community edition version of UI
- `@postgres.ai/shared` - common modules

## UI Development Documentation

Detailed information about UI development has been moved to the main [CONTRIBUTING.md](../CONTRIBUTING.md#ui-development) file. Please refer to that document for:

- How to operate UI packages
- Platform UI development setup
- Building CE and Platform versions
- CI pipelines for UI code
- Handling vulnerabilities
- TypeScript migration

This centralized approach ensures all development information is maintained in one place.

# Changelog


## [0.2.0] - 2020-02-06

In Database Lab 0.2.0, all the components now run in containers. To start using Database Lab is now only needed to install Docker and ZFS, everything else is inside containers. Customer PostgreSQL containers are supported.

Important links:

- Project repository: https://gitlab.com/postgres-ai/database-lab
- Issue tracker (for bugs reports, feature proposals): https://gitlab.com/postgres-ai/database-lab/issues
- Tutorial: https://postgres.ai/docs/database-lab/1_tutorial
- Community Slack (English): https://postgresteam.slack.com/archives/CSXFWTD0Q

### Server
- All components of Database Lab server now run on Docker containers.
- Support custom Postgres containers.
- Ability to customize Postgres user used for management (`postgres` is default).
- Verification token flag changed from "-v" to "-t".
- Internal: use constants for response errors.
- Various imrovements of the snapshotting script.

### API
- Synchronous mode for clone methods (now default).
- Parameter `name` furrly removed
- Parameter `id` can now be set by user (optionally).
- Internal: added Metadata to Clone model.

### Client CLI
- Output improved.
- Show "unchanged" message in "config update" command if params match the current config.
- Added client installation script.

### Documentaion
- Added Swagger specification.
- Added Postman collection.
- Updated README.
- Tutorial fully reworked.
- New pages/sections in the docs: Overview, Workflow, Requirements, CLI Reference, Q&A.
- Added "ci-example" repository.

### CI
- Build and push Docker images to Docker Hub and Gitlab Registry.
- Build and upload binaries for linux, alpine, darwin to GitLab.

### Links
- Project repository: https://gitlab.com/postgres-ai/database-lab
- Issue tracker (for bugs reports, feature proposals): https://gitlab.com/postgres-ai/database-lab/issues
- SQL optimization usage example: https://gitlab.com/postgres-ai/joe
- Database changes verification usage example: https://gitlab.com/postgres-ai/ci-example
- Tutorial: https://postgres.ai/docs/database-lab/1_tutorial
- Slack (English): https://postgresteam.slack.com/archives/CSXFWTD0Q
- Telegram (Russian): https://t.me/databaselabru


## [0.1.0] - 2020-01-23
### Features
- Superfast cloning of large databases
- REST API
- Client SDK
- CLI tool
- Auto-deletion of clones after some period of inactivity (by default, 2 hours)
- Protection from deletion: a clone can be marked as "protected", which means that it cannot be deleted, neither manually nor automatically
- Postgres versions 9.6, 10, 11, 12 are supported
- Postgres clones are running in Docker containers. Custom Docker images are supported, to allow additional extensions, libraries, custom Postgres builds
- Custom postgresql.conf addition and pg_hba.conf replacement to control settings of Postgres clones
- For security, all Postgres users are disabled, and a new one is created (configure at clone request time)

### Known issues and future work
1. Currently, to install Database Lab, a lot of software needs to be installed: Docker, Go, ZFS, Postgres, more. See Tutorial. Tested on Ubuntu 18.04. In the upcoming Database Lab versions, we will move everything to containers, which means that only Docker will be a requirement, and any modern Linux will be supported.
1. If a Database Lab instance is restarted, all the clones get deleted. We plan to implement persistent clones in the future versions, to keep all existing clones during restarts.

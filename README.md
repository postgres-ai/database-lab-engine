# Database Lab - thin database clones for faster development

<img src="./assets/db-lab.png" align="right" border="0" />

Database Lab Engine is an open-source technology that allows blazing-fast cloning of Postgres databases of any size in seconds. This helps solve many problems such as:
- help build dev/QA/staging environments involving full-size production-like databases,
- provide temporary full-size database clones for SQL query analysis optimization (see [Joe bot](https://gitlab.com/postgres-ai/joe)),
- automatically verify database migrations (DB schema changes) and massive data operations to avoid downtime and performance bottlenecks.

As an example, cloning of 10 TiB PostgreSQL database takes less than 2 seconds.

Read more at [Postgres.ai](https://postgres.ai) and in [the Docs](https://postgres.ai/docs).

> Please support the project giving a GitLab star! It's on [the main page](https://gitlab.com/postgres-ai/database-lab), at the upper right corner:
>
> ![Add a star](./assets/star.gif)

## Installation
See the Database Lab tutorials:
- [Database Lab tutorial for any PostgreSQL database](https://postgres.ai/docs/tutorials/database-lab-tutorial).
- [Database Lab tutorial for Amazon RDS](https://postgres.ai/docs/tutorials/database-lab-tutorial-amazon-rds),

The latest stable Docker images are published in [the Docker Hub repository](https://hub.docker.com/repository/docker/postgresai/dblab-server).

## Client CLI
### Installation
Install Database Lab client CLI on your Linux machine (e.g., Ubuntu):
```bash
curl https://gitlab.com/postgres-ai/database-lab/-/raw/master/scripts/cli_install.sh | bash
```

### Activate CLI autocompletion:
```bash
sudo curl https://gitlab.com/postgres-ai/database-lab/-/raw/master/scripts/bash_autocomplete --output /etc/bash_completion.d/dblab 
source /etc/bash_completion.d/dblab
```

### How to use CLI
- [How to install and initialize Database Lab CLI](https://postgres.ai/docs/guides/cli/cli-install-init)
- [Database Lab Client CLI reference (dblab)](https://postgres.ai/docs/database-lab/cli-reference)

## Docker Hub
- [Server](https://hub.docker.com/repository/docker/postgresai/dblab-server)
- [CLI client](https://hub.docker.com/repository/docker/postgresai/dblab)
- [Custom Postgres images](https://hub.docker.com/repository/docker/postgresai/extended-postgres)

## References
- [Database Lab Engine configuration reference](https://postgres.ai/docs/database-lab/config-reference)
- [API reference](https://postgres.ai/swagger-ui/dblab/)
- [CLI reference](https://postgres.ai/docs/database-lab/cli-reference)

## Development
Open [an Issue](https://gitlab.com/postgres-ai/database-lab/-/issues) to discuss ideas, open [a Merge Request](https://gitlab.com/postgres-ai/database-lab/-/merge_requests) to propose a change.

See our [GitLab Container Registry](https://gitlab.com/postgres-ai/database-lab/container_registry) to find the images built for development branches.
<!-- TODO: SDK docs -->
<!-- TODO: Contribution guideline -->

### Development requirements
1. Install `golangci-lint`: https://github.com/golangci/golangci-lint#install

## Have a question?
- Check our [Q&A](https://postgres.ai/docs/questions-and-answers)
- or join our Community (links below)

## Community
- [Community Slack (English)](https://database-lab-team-slack-invite.herokuapp.com/)
- [Telegram (Russian)](https://t.me/databaselabru)
- [Twitter](https://twitter.com/Database_Lab)


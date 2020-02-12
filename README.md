# Database Lab - Thin database clones for faster development

<img src="./assets/db-lab.png" align="right" border="0" />
Database Lab allows superfast cloning of large databases to solve the following problems:

- help build development and testing environments involving full-size database,
- provide temporary full-size database clones for SQL query optimization (see also:
[Joe bot](https://gitlab.com/postgres-ai/joe), which works on top of Database Lab),
- help verify database migrations (DB schema changes) and massive data operations.

As an example, the cloning of 10 TiB PostgreSQL database takes less than 2 seconds.

In rapidly developing businesses, excellent production health requires powerful non-production environments.
With Database Lab, provisioning of multi-terabyte database clones doesn't imply much
waiting time or extra budgets spent anymore. Therefore, Database Lab gives necessary power to developers, DBAs,
and QA engineers, boosting development and testing processes.


## Status

The project is in its early stage. However, it is already being extensively used
in some teams in their daily work. Since production is not involved, it is
quite easy to try and start using it.

Please support the project giving a GitLab star (it's on [the main page](https://gitlab.com/postgres-ai/database-lab),
at the upper right corner):

![Add a star](./assets/star.gif)

To discuss Database Lab, try a demo, or ask any questions,
[join our community Slack](https://database-lab-team-slack-invite.herokuapp.com/).


## Usage examples
- Perform SQL optimization in a convenient way (see [postgres-ai/joe](https://gitlab.com/postgres-ai/joe))
- Check database schema changes (database migrations) on full-sized database clones using Database Lab in CI (see [postgres-ai/ci-example](https://gitlab.com/postgres-ai/ci-example))


## Workflow overview and requirements

**TL;DR:** you need:
- any machine with a separate disk that is big enough to store a single copy of your database,
- Linux with Docker and ZFS,
- initial copy of your Postgres database.

Details:
- for each Database Lab instance, a separate machine (either physical or virtual) is needed,
- both on-premise and cloud setups are possible,
- for each Postgres cluster (a single Postgres server with databases), a separate Database Lab instance is required,
- the machine needs to have a separate disk partition with size enough to store the target Postgres directory (PGDATA),
- any modern Linux is supported,
- Docker needs to be installed on the machine,
- currently, you need to take care yourself of the initial copying of the database to this disk ("thick cloning" stage),
use pg_basebackup, restoration from an archive (such as WAL-G, Barman, pgBackRest or any), or dump/restore (the only way
supported for RDS, until AWS guys decide to allow replication connections),
- upon request, Database Lab will do "thin cloning" of PGDATA, providing fully independent writable
Postgres clones to users;
- currently, the only technology supported for thin cloning is ZFS,
so [ZFS on Linux](https://zfsonlinux.org/) needs to be installed on the machine,
- however, it is easy to extend and add, say, LVM or Ceph - please write us if you
need it; also, contributions are highly welcome).


## Server installation and setup
See [detailed tutorial](https://postgres.ai/docs/database-lab/1_tutorial)
in our documentation.

For stable Docker images see [postgresai/dblab-server](https://hub.docker.com/repository/docker/postgresai/dblab-server) repository on DockerHub.


## Client CLI
### Installation
Install Database Lab client CLI on a Linux architecture (e.g., Ubuntu):
```bash
curl https://gitlab.com/postgres-ai/database-lab/-/raw/master/cli-install.sh | bash
```

Also, binaries available for download: [Alpine](https://gitlab.com/postgres-ai/database-lab/-/jobs/artifacts/master/browse?job=build-binary-alpine), [Other](https://gitlab.com/postgres-ai/database-lab/-/jobs/artifacts/master/browse?job=build-binary-generic).


### Usage
See the full client CLI reference [here](https://postgres.ai/docs/database-lab/6_cli_reference).

Once you have Database Lab server(s) intalled, and client CLI installed on your machine,
initialize CLI and start communicating with the Database Lab server(s).

### Initialize CLI tool
```bash
dblab init \
  --environment_id=tutorial \
  --url=http://$IP_OR_HOSTNAME:3000 \
  --token=secret_token
```

### Check connection availability
Access your Database Lab instance and check its status.
```bash
dblab instance status
```

### Request clone creation
When your Database Lab instance is up and running you can use it to create thin
clones, work with them, delete the existing clones, and see the list of
existing clones. To create a thin clone, you need to execute a `dblab clone create`
and fill all the required options, as illustrated below:

```bash
dblab clone create \
  --username dblab_user_1 \
  --password secret_password \
  --id my_first_clone
```

After a few seconds, if everything is configured correctly, you will see
that the clone is ready to be used:
```json
{
    "id": "botcmi54uvgmo17htcl0",
    "snapshot": {
        "id": "dblab_pool@initdb",
        ...
    },
    "status": {
        "code": "OK",
        "message": "Clone is ready to accept Postgres connections."
    },
    "db": {
        "connStr": "host=localhost port=6000 user=dblab_user_1",
        "host": "localhost",
        "port": "6000",
        "username": "dblab_user_1",
        "password": ""
    },
    ...
}
```


### Connect
Now you can work with this clone using any PostgreSQL client, for example `psql`:
```bash
PGPASSWORD=secret_password \
  psql "host=${IP_OR_HOSTNAME} port=6000 user=dblab_user_1 dbname=test" \
  -c '\l+'
```


## References
- [API reference](https://postgres.ai/swagger-ui/dblab/)
- [CLI reference](https://postgres.ai/docs/database-lab/6_cli_reference)


## Have questions?
- Check our [Q&A](https://postgres.ai/docs/get-started#qa)
- or join our community (links below)


## Community
- [Community Slack (English)](https://database-lab-team-slack-invite.herokuapp.com/)
- [Telegram (Russian)](https://t.me/databaselabru)


## Development
See our [GitLab Container Registry](https://gitlab.com/postgres-ai/database-lab/container_registry) for develop Docker images.

### Requirements
1. Install `golangci-lint`: https://github.com/golangci/golangci-lint#install
<!-- TODO: SDK docs -->
<!-- TODO: Contribution guideline -->

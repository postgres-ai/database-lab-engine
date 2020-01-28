# Database Lab - Thin clones for faster development
Boost your development process

<div align="center">
    ![Database Lab logo](./assets/db-lab.png)
</div>

Database Lab allows superfast cloning of large databases to solve the following
problems:
- help build development and testing environments for development and testing
involving full-size database,
- provide temporary full-size database clones for SQL query optimization
(see also: [Joe bot](https://gitlab.com/postgres-ai/joe), which works on top of Database Lab),
- help verify database migrations (DB schema changes) and massive data operations.

As an example, the cloning of 10 TiB PostgreSQL database takes less than 2 seconds.

## Getting started
See [detailed tutorial](https://postgres.ai/docs/database-lab/1_tutorial)
in our documentation.

## CLI Install

*Currently, there are no ready-to-use binaries or a Docker image. The setup
is to be done using the source code.*

1. Get the source code: `git clone https://gitlab.com/postgres-ai/db-lab.git`.
1. Golang is required.
    - Ubuntu: In some cases, standard Ubuntu package might not work. See
https://gitlab.com/postgres-ai/db-lab/snippets/1918769.
    - On macOS: `brew install golang`
1. Clone Database Lab repo and build it:
``` bash
git clone https://gitlab.com/postgres-ai/database-lab.git
cd ./database-lab
make all
```


## Usage
### Initialize CLI tool
```bash
./bin/dblab init --environment_id dev1 --url http://HOST --token TOKEN
```

### Check connection availability
Access your Database Lab instance and check its status.
```bash
./bin/dblab instance status
```

If the Database Lab instance is functioning normally, we will get the status
code `OK`, and the response will have the following format:
```json
{
  "status": {
    "code": "OK",
    "message": "Instance is ready"
  },
  ...
}
```

### Request clone creation
When your Database Lab instance is up and running you can use it to create thin
clones, work with them, delete the existing clones, and see the list of
existing clones. To create a thin clone, you need to execute a `dblab clone create`
and fill all the required options, as illustrated below:

```bash
./bin/dblab clone create --name Clone1 --username dblab_user1 --password secret
```

We will get clone ID and status `CREATING`, we should execute consequential
`./bin/dblab instance status` and wait until status became `OK`.
```json
{
  "id": "bo200eumq8of32ck5e2g",
  "name": "Clone1",
  "status": {
    "code": "OK",
    "message": "Clone is ready to accept connections."
  },
  "db": {
    "host": "internal_host",
    "port": "6000",
    "username": "dblab_user1"
  },
  ...
}
```

### Connect
When the status is `OK` we are ready to connect to the clone's Postgres
database. For example, using `psql`:
```bash
psql -h internal_host -p 6000 -U dblab_user1 # will ask for a password unless it's set in either PGPASSWORD or .pgpass
```

## Community
- Slack (English): https://postgresteam.slack.com/archives/CSXFWTD0Q
- Telegram (Russian): https://t.me/databaselabru

## Development requirements

1. Install golangci-lint: https://github.com/golangci/golangci-lint#install

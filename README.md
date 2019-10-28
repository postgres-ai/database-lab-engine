# Database Lab - Thin clones for faster development
Boost your backend development process

Provide developers access to experiment on automatically provisioned
production-size DB testing replica. Joe will provide recommendations
for query optimization and the ability to rollback.

## Install Software

*Currently, there are no ready-to-use binaries or a Docker image. The setup
is to be done using the source code.*

1. Get the source code: `git clone https://gitlab.com/postgres-ai/joe.git`.
1. It is not recommended to use HTTP: all the traffic from Slack to Joe should
be encrypted. It can be achieved by using NGINX with a free Let's Encrypt
certificate (Ubuntu: https://gitlab.com/postgres-ai/joe/snippets/1876422).
1. Golang is required.
    - Ubuntu: In some cases, standard Ubuntu package might not work. See
https://gitlab.com/postgres-ai/joe/snippets/1880315.
    - On macOS: `brew install golang`
1. If needed (when working in "local" mode), install ZFS (Ubuntu:
https://gitlab.com/postgres-ai/joe/snippets/1880313).

### ZFS Store
1. Create a ZFS store with a clone of
the production Postgres database (e.g. using WAL-E, WAL-E or Barman archive).
1. Shutdown Postgres, create a new ZFS snapshot
(`sudo zfs snapshot -r  zpool@db_state_1`) and remember its name. It will
be needed for further configuration (`initialSnapshot` option in 
`config/provisioning.yaml`).
1. Start Postgres.

### Deploy
Deploy Joe instance in your infrastructure. You would need to:
1. Create `config/provisioning.yaml` and `config/envs/MY_ENV.sh` with desired configuration (see samples in corresponding directories).
1. Make a publicly accessible HTTP(S) server port specified in the configuration for Slack Events Request URL.
1. Build and run Joe `ENV=MY_ENV bash ./do.sh run` (or, with log: `ENV=MY_ENV bash ./do.sh run 2>&1 | tee -a joe.log`).

Unless being run in the "local" mode, Joe will automatically provision AWS EC2
or GCP GCE instance of Postgres.

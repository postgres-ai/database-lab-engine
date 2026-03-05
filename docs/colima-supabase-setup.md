# DBLab with Colima (ARM64/Apple Silicon)

This guide explains how to run DBLab Engine on macOS with Apple Silicon using Colima.

## Prerequisites

- macOS with Apple Silicon (M1/M2/M3/M4)
- [Colima](https://github.com/abiosoft/colima) installed
- Docker CLI installed
- Go 1.24+ (for building from source)

## Known Limitations

### ARM64 Docker Images

The official DBLab images are currently **amd64 only**:
- `postgresai/extended-postgres:*` - amd64 only
- `postgresai/ce-ui:*` - amd64 only

For ARM64, you need to build images from source (see below).

## Setup Steps

### 1. Start Colima and Set Up ZFS

Colima does not include ZFS by default. Use the provided init script to install ZFS utilities and create the pool:

```bash
colima start

# Copy and run the ZFS init script inside Colima
colima ssh < engine/scripts/init-zfs-colima.sh
```

The script installs `zfsutils-linux`, creates a 5 GB virtual disk, and sets up a ZFS pool (`dblab_pool`) with datasets at `/var/lib/dblab/dblab_pool`.

If you need to customize the pool size or dataset names, edit the variables at the top of `engine/scripts/init-zfs-colima.sh` before running it.

### 2. Build ARM64 Images

Make sure your Docker CLI is configured to use Colima's Docker daemon:

```bash
docker context use colima
```

All `docker` commands below run against Colima, so no image transfer is needed.

#### DBLab Server

From the repository root:

```bash
cd engine
GOOS=linux GOARCH=arm64 make build
make build-image
```

This builds the server binary for ARM64 and creates a Docker image tagged `dblab_server:local`.

#### PostgreSQL Image

The `postgresai/extended-postgres` images are amd64 only. For ARM64, create a minimal Postgres image.

In the `engine/` directory, create `Dockerfile.dblab-postgres-arm64`:

```dockerfile
FROM postgres:17-bookworm

ENV PG_UNIX_SOCKET_DIR=/var/run/postgresql
ENV PG_SERVER_PORT=5432

RUN apt-get update && apt-get install -y --no-install-recommends sudo \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /var/lib/postgresql/17/

RUN echo '#!/bin/bash' > /pg_start.sh && chmod a+x /pg_start.sh \
    && echo 'chown -R postgres:postgres ${PGDATA} ${PG_UNIX_SOCKET_DIR}' >> /pg_start.sh \
    && echo 'sudo -Eu postgres /usr/lib/postgresql/17/bin/postgres -D ${PGDATA} -k ${PG_UNIX_SOCKET_DIR} -p ${PG_SERVER_PORT} >& /proc/1/fd/1' >> /pg_start.sh \
    && echo '/bin/bash -c "trap : TERM INT; sleep infinity & wait"' >> /pg_start.sh

CMD ["/pg_start.sh"]
```

```bash
docker build -f Dockerfile.dblab-postgres-arm64 \
  --platform linux/arm64 \
  -t dblab-postgres:17-arm64 .
```

#### DBLab CE UI (optional)

From the repository root:

```bash
cd ..
docker build -f ui/packages/ce/Dockerfile \
  --platform linux/arm64 \
  -t dblab-ce-ui:arm64 .
```

### 3. Create DBLab Configuration

Create `engine/configs/server.yml` based on one of the example configs in `engine/configs/`. A minimal example:

```yaml
server:
  verificationToken: "your_token_here"
  host: "0.0.0.0"
  port: 2345

embeddedUI:
  enabled: false  # We'll run UI separately for ARM64

global:
  engine: postgres
  debug: true
  database:
    username: postgres
    dbname: postgres

poolManager:
  mountDir: /var/lib/dblab/dblab_pool
  dataSubDir: ""
  clonesMountSubDir: clones
  socketSubDir: sockets
  observerSubDir: observer
  preSnapshotSuffix: "_pre"
  selectedPool: "dataset_1"

databaseContainer:
  dockerImage: "dblab-postgres:17-arm64"
  containerConfig:
    "shm-size": 256mb

provision:
  portPool:
    from: 6000
    to: 6010
  useSudo: false
  keepUserPasswords: true
  cloneAccessAddresses: "127.0.0.1"

retrieval:
  refresh:
    timetable: ""
  jobs:
    - physicalSnapshot
  spec:
    physicalSnapshot:
      options:
        skipStartSnapshot: false
        promotion:
          enabled: false

cloning:
  accessHost: "localhost"
  maxIdleMinutes: 60

platform:
  enableTelemetry: false
```

### 4. Initialize PostgreSQL Data on ZFS

Start a PostgreSQL container to initialize data on the ZFS dataset:

```bash
docker run -d --name pg-init \
  -e POSTGRES_PASSWORD=postgres \
  -v /var/lib/dblab/dblab_pool/dataset_1:/var/lib/postgresql/data \
  postgres:17-bookworm

# Wait for initialization to complete
docker logs -f pg-init
# Press Ctrl+C once you see "database system is ready to accept connections"

# Stop and remove the init container
docker stop pg-init && docker rm pg-init

# Create the initial snapshot
colima ssh -- sudo zfs snapshot dblab_pool/dataset_1@initial_data
```

### 5. Start DBLab Server

Use `rshared` mount propagation so that ZFS clones are visible inside child containers.

First, copy the config into the Colima VM. From the `engine/` directory:

```bash
colima ssh -- mkdir -p /var/lib/dblab/configs
colima ssh -- tee /var/lib/dblab/configs/server.yml < configs/server.yml > /dev/null
```

Then start the server:

```bash
docker run -d --name dblab-server \
  --privileged \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /var/lib/dblab:/var/lib/dblab:rshared \
  -v /var/lib/docker:/var/lib/docker \
  -v /var/lib/dblab/configs:/home/dblab/configs \
  -e HOSTNAME=dblab-server \
  -p 2345:2345 \
  dblab_server:local
```

### 6. Start DBLab UI (optional)

```bash
docker run -d --name dblab-ce-ui \
  -e DLE_HOST=host.docker.internal \
  -e DLE_PORT=2345 \
  -p 2346:80 \
  --add-host=host.docker.internal:host-gateway \
  dblab-ce-ui:arm64
```

### 7. Use DBLab CLI

```bash
cd engine && make build-client

dblab init \
  --url http://localhost:2345 \
  --token your_token_here \
  --environment-id local

dblab clone create --id dev1 --username postgres --password 'SecurePass123!'

dblab clone list

# Connect to a clone
psql -h localhost -p 6000 -U postgres -d postgres
```

## Supabase Integration

To use DBLab with a Supabase-managed PostgreSQL instance:

1. Set the database user to `supabase_admin` (Supabase superuser) in the config:

   ```yaml
   global:
     database:
       username: supabase_admin
   ```

2. Initialize data using the Supabase Postgres image instead of stock Postgres:

   ```bash
   docker run -d --name supabase-pg-init \
     -e POSTGRES_PASSWORD=postgres \
     -v /var/lib/dblab/dblab_pool/dataset_1:/var/lib/postgresql/data \
     -p 5433:5432 \
     public.ecr.aws/supabase/postgres:17.6.1.064
   ```

3. Set `dataSubDir: ""` in poolManager since Supabase puts data at the dataset root.

## Troubleshooting

### Clone containers cannot see data

Ensure dblab-server is started with `-v /var/lib/dblab:/var/lib/dblab:rshared` for mount propagation.

### "permission denied to alter role"

The database user in your config needs superuser privileges. For Supabase, use `supabase_admin`.

### UI shows JSON parse errors

The UI calls some API endpoints directly (without the `/api/` prefix). Make sure the nginx proxy configuration includes all required endpoint locations.

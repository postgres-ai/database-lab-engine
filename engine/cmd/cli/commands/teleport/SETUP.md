# Teleport integration — setup guide

This document covers the prerequisites and configuration required to run
`dblab teleport serve` alongside a DBLab Engine instance.

## Architecture overview

```
┌──────────────┐   webhooks    ┌──────────────────┐   tctl    ┌──────────────────┐
│  DBLab Engine │─────────────▶│ dblab teleport   │─────────▶│ Teleport Auth    │
│  (Docker)     │              │ serve (sidecar)   │          │ Server           │
└──────────────┘              └──────────────────┘          └──────────────────┘
       │                                                           │
       │ clone containers                                          │
       ▼                                                           ▼
┌──────────────┐              ┌──────────────────┐          ┌──────────────────┐
│  Clone PG    │◀─────────────│ Teleport DB Agent │◀─────────│ tsh proxy db    │
│  (port 6000) │   proxied    │ (db_service)      │  tunnel  │ (end user)      │
└──────────────┘              └──────────────────┘          └──────────────────┘
```

The sidecar:
- Receives `clone_create` / `clone_delete` webhooks from DBLab Engine
- Calls `tctl create` / `tctl rm` to register/deregister Teleport DB resources
- Runs startup reconciliation to catch missed events

The sidecar does **not** proxy database connections. A separate Teleport agent
with `db_service` enabled handles the actual proxying.

---

## Prerequisites

### 1. Bot role permissions

The bot role must be created **before** generating the bot identity, because the
identity captures the role's permissions at generation time.

```yaml
kind: role
version: v7
metadata:
  name: dblab-bot
spec:
  allow:
    db_labels:
      '*': '*'
    db_names: ['*']
    db_users: ['*']
    rules:
    - resources: [db, db_server]
      verbs: [list, create, read, update, delete]
    - resources: [app, app_server]
      verbs: [list, create, read, update, delete]
```

Apply with `tctl create -f dblab-bot-role.yaml`.

### 2. Teleport bot identity

The sidecar authenticates to the Teleport Auth Server using a bot identity file.
Create a bot and generate the identity:

> **Important:** The role from §1 must already exist before this step. The bot
> identity captures the role's permissions at generation time. If the role is
> created or updated after the identity is generated, regenerate the identity
> to pick up the new permissions.

**Self-hosted Teleport:**
```bash
tctl bots add dblab-sidecar --roles=dblab-bot
tctl auth sign --format=tls --user=bot-dblab-sidecar -o /etc/teleport/dblab-identity
```

**Teleport Cloud** (`tctl auth sign` is not available):
```bash
tctl bots add dblab-sidecar --roles=dblab-bot
# Use the token from the output above
tbot start --oneshot \
  --token=<TOKEN> \
  --proxy-server=yourcluster.teleport.sh:443 \
  --join-method=token \
  --data-dir=/etc/teleport/bot-data \
  --destination-dir=/etc/teleport/bot-dest
# The identity file is at /etc/teleport/bot-dest/identity
```

### 3. Teleport database agent

The sidecar only registers DB resources via `tctl` — it does **not** proxy
connections. A Teleport agent must run on the DBLab host with `db_service`
enabled and dynamic resource matching:

```yaml
# /etc/teleport.yaml (on the DBLab host)
db_service:
  enabled: true
  resources:
  - labels:
      dblab: "true"
```

This tells the agent to proxy connections for any DB resource with the
`dblab: "true"` label (which the sidecar sets automatically).

### 4. User role for database access

Teleport users who need to connect to DBLab clones need a role granting
database access:

```yaml
kind: role
version: v7
metadata:
  name: dblab-user
spec:
  allow:
    db_labels:
      dblab: "true"
    db_names: ['*']
    db_users: ['*']
```

To gate access on your own taxonomy instead of `dblab: "true"`, attach custom
labels with `--label` and match on those — see [Resource Labels](#resource-labels).

### 5. SSL/TLS for Postgres clones

Teleport **always** initiates TLS to backend databases, even when `tls.mode: insecure`
is set (that only skips certificate *verification*, it does not skip TLS itself).

DBLab clones must have SSL enabled with valid cert/key files.

**Generate self-signed certs:**
```bash
openssl req -new -x509 -days 3650 -nodes \
  -out /etc/dblab/certs/server.crt \
  -keyout /etc/dblab/certs/server.key \
  -subj "/CN=dblab-clone"

# Certs must be owned by postgres user (uid 999 inside the container)
chown 999:999 /etc/dblab/certs/server.crt /etc/dblab/certs/server.key
chmod 600 /etc/dblab/certs/server.key
```

**Export the Teleport DB CA certificate:**

PostgreSQL needs the Teleport CA certificate to verify client certificates
presented by the Teleport DB agent. Without `ssl_ca_file`, the `cert` auth
method in `pg_hba.conf` fails with "root certificate store not available".

```bash
tctl auth export --type=db-client > /etc/dblab/certs/teleport-ca.crt
chown 999:999 /etc/dblab/certs/teleport-ca.crt
```

**Add to server.yml:**
```yaml
databaseConfigs: &db_configs
  configs:
    ssl: "on"
    ssl_cert_file: "/var/lib/postgresql/cert/server.crt"
    ssl_key_file: "/var/lib/postgresql/cert/server.key"
    ssl_ca_file: "/var/lib/postgresql/cert/teleport-ca.crt"
```

> **Note:** After adding or changing `databaseConfigs`, a data refresh is
> required. These settings are applied during snapshot creation and baked into
> `postgresql.dblab.snapshot.conf`. Existing snapshots are not affected.

### 6. pg_hba.conf — certificate authentication

Starting with DBLab Engine 4.1.0, the default `pg_hba.conf` includes a `hostssl ... cert`
rule that enables Teleport certificate authentication out of the box:

```
local all all trust
hostssl all all 0.0.0.0/0 cert
host all all 0.0.0.0/0 md5
```

No custom `pg_hba.conf` or volume mount is required for Teleport.

**How rule evaluation works:**
- PostgreSQL evaluates `pg_hba.conf` top to bottom and uses the first matching rule.
- `hostssl ... cert` matches SSL connections and requires a client certificate.
  Teleport always connects over SSL with a client certificate, so this rule handles
  Teleport connections.
- `host ... md5` matches both SSL and non-SSL connections. Non-SSL password
  connections (e.g., `sslmode=disable`) skip the `hostssl` rule and match here.

> **Note:** SSL connections *without* a client certificate (e.g., `sslmode=require`
> with password auth only) will be matched by the `hostssl ... cert` rule and
> rejected. Clients that do not use Teleport should connect with `sslmode=disable`
> or `sslmode=prefer` (which falls back to non-SSL when cert auth fails).

### 7. Volume mounting for certs

Clone containers only inherit DBLab Engine container volumes whose source is under
`poolManager.mountDir`. For SSL certs stored outside the pool, use
`containerConfig`:

```yaml
databaseContainer: &db_container
  dockerImage: "postgresai/extended-postgres:16"
  containerConfig:
    "shm-size": 1gb
    volume: "/etc/dblab/certs:/var/lib/postgresql/cert:ro"
```

**Important:** Cert files on the host must have uid 999 ownership *before*
DBLab Engine starts, because the postgres user inside the container runs as uid 999.

### 8. Webhook URL — Docker networking

DBLab Engine runs inside Docker, so `localhost:9876` from within the Engine
container resolves to the container itself, not the host.

Options:
- Use `host.docker.internal:9876` (Docker Desktop / Docker 20.10+)
- Use the Docker bridge IP (typically `172.17.0.1:9876`)
- Run the sidecar in the same Docker network as DBLab Engine

The sidecar should listen on `0.0.0.0:9876` (not `localhost:9876`) if
containers need to reach it:

```bash
dblab teleport serve \
  --listen-addr 0.0.0.0:9876 \
  ...
```

**Configure webhooks in server.yml:**
```yaml
webhooks:
  hooks:
    - url: "http://host.docker.internal:9876/teleport-sync"
      secret: "your-webhook-secret"
      trigger:
        - clone_create
        - clone_delete
```

---

## Full server.yml example (Teleport-relevant sections)

```yaml
databaseContainer: &db_container
  dockerImage: "postgresai/extended-postgres:16"
  containerConfig:
    "shm-size": 1gb
    volume: "/etc/dblab/certs:/var/lib/postgresql/cert:ro"

databaseConfigs: &db_configs
  configs:
    ssl: "on"
    ssl_cert_file: "/var/lib/postgresql/cert/server.crt"
    ssl_key_file: "/var/lib/postgresql/cert/server.key"
    ssl_ca_file: "/var/lib/postgresql/cert/teleport-ca.crt"

webhooks:
  hooks:
    - url: "http://host.docker.internal:9876/teleport-sync"
      secret: "your-webhook-secret"
      trigger:
        - clone_create
        - clone_delete
```

## Running the sidecar

```bash
dblab teleport serve \
  --environment-id production \
  --teleport-proxy teleport.example.com:3025 \
  --teleport-identity /etc/teleport/dblab-identity \
  --listen-addr 0.0.0.0:9876 \
  --dblab-url http://localhost:2345 \
  --dblab-token "$DBLAB_TOKEN" \
  --webhook-secret "$WEBHOOK_SECRET" \
  --label environment=production \
  --label db-type=main \
  --label service=dblab
```

## Resource labels

Every Teleport `db` and `app` resource the sidecar creates carries a set of
labels. Some are managed by the sidecar and cannot be overridden; the rest are
supplied by the operator to match an existing Teleport taxonomy.

**Managed (reserved) labels — always set:**

| Label | Value | Purpose |
|-------|-------|---------|
| `dblab` | `"true"` | Marks the resource as DBLab-managed; used by the agent matcher and user roles |
| `dblab_instance` | `<environment-id>` | Owning DBLab instance; used internally to keep reconciliation isolated per instance |
| `clone_id` | clone ID | DB resources only |
| `dblab_user` | authenticated user (full email address) | DB resources only, set when clone binding is enabled and the creator used a personal token |
| `environment` | `<environment-id>` | Set **only** when no custom `environment` label is provided (backwards compatibility) |

> **Important:** Each DBLab instance must use a **unique** `--environment-id`.
> It becomes the `dblab_instance` ownership label, and instances that share one
> would reconcile each other's resources.

**Custom labels — `--label key=value` (repeatable):**

Use `--label` to attach any additional labels so DBLab clones fit the same
access taxonomy as other database resources in your cluster — keys and values
are entirely operator-defined. Labels can also be supplied via the
`TELEPORT_LABELS` environment variable (comma-separated, e.g.
`TELEPORT_LABELS=environment=production,db-type=main`). Reserved keys
(`dblab`, `dblab_instance`, `clone_id`, `dblab_user`) are rejected at startup.

Because instance ownership is tracked by the dedicated `dblab_instance` label,
the `environment` label is free for operator use. Multiple DBLab instances can
therefore share the same `environment` value (e.g. `production`) and be
distinguished by `db-type` while each `--environment-id` keeps reconciliation
isolated:

```bash
# instance serving the main OLTP database
dblab teleport serve --environment-id production-main \
  --label environment=production --label db-type=main --label service=dblab ...

# instance serving the analytics database
dblab teleport serve --environment-id production-analytics \
  --label environment=production --label db-type=analytics --label service=dblab ...
```

A user/bot role can then gate access on the stable labels, with the rest as
wildcards:

```yaml
spec:
  allow:
    db_labels:
      service: ["dblab"]
      environment: ["*"]
      db-type: ["*"]
    app_labels:
      service: ["dblab"]
      environment: ["*"]
      db-type: ["*"]
```

> **Note:** Teleport requires the resource to carry **every** label key named in
> a role's `db_labels`/`app_labels`. If a role lists a key (e.g. `writable`) that
> the sidecar does not set, add it with `--label writable=readwrite`, otherwise
> access is denied.

## Per-user clone access

By default, every Teleport user who holds a role granting access to DBLab
resources can connect to **any** clone. To make access per-user, the Engine can
attach a trusted `dblab_user` label to each clone, derived from the
**authenticated user's identity** rather than from any client-supplied value.

### Enable binding in server.yml

```yaml
platform:
  enablePersonalTokens: true   # required — identity comes from the personal token
  bindClonesToUser: true
```

When `bindClonesToUser` is enabled, a clone created with a **personal per-user
token** is labeled `dblab_user: <email>` — the authenticated user's full email
address, matching Teleport's `external.email`. The clone's Postgres username
(`db.username`) is **not** changed, so existing connection strings, Joe, and CI
automation keep working.

Clones created with the shared `verificationToken` (for example CI pipelines or
Joe) carry **no** `dblab_user` label. They are created normally — not rejected —
but are not reachable through a per-user role that matches on `dblab_user`; grant
those callers access through a broader role instead.

A trusted proxy that authenticates with the shared `verificationToken` on behalf
of a known user (for example the PostgresAI Platform serving console requests)
can assert the acting user by sending the `X-Forwarded-User-Email` header. The
Engine then labels the clone exactly as if that user had used a personal token.
The header is ignored on personal-token requests and when authorization is
disabled; asserting an identity grants the caller nothing beyond what the shared
token already allows.

### Restrict access with a per-user role

```yaml
kind: role
version: v7
metadata:
  name: dblab-self-access
spec:
  allow:
    db_labels:
      dblab: ['true']
      dblab_user: ['{{external.email}}']
    db_names: ['*']
    db_users: ['*']
```

With this role a user can reach only the clones labeled with their own email,
e.g. `jsmith@acme.com` reaches resources labeled `dblab_user: jsmith@acme.com`.

### Limitations

- **Two identity sources must agree.** The label is the full email the
  PostgresAI Platform returns at clone-create time, while Teleport matches
  against `external.email` from its own SSO/IdP at connect time. Both sides
  preserve case, and the `dblab_user` label is case-sensitive, so the two emails
  must match **exactly, including case** (`JSmith@acme.com` and `jsmith@acme.com`
  are different). If they differ, the user is denied access to their own clone
  (it fails closed). Make sure the Platform and the Teleport IdP emit the same
  address for each user. This includes the **domain**: because the full address
  is compared, a source that normalizes the domain case on only one side
  (`user@Acme.io` vs `user@acme.io`) also fails closed, even though DNS treats
  the two as the same mailbox.
- Email addresses that cannot be represented as a Teleport label value
  (those containing `+` or other characters outside `[a-zA-Z0-9._@-]`) produce an
  **unlabeled** clone with a warning in the Engine log — the label is never
  silently rewritten, so when a label is present it always equals Teleport's
  value. Such users need a broader role to reach their clones.
- Clones created before binding was enabled, or with the shared token, have no
  `dblab_user` label; recreate them with a personal token to apply it.
- Upgrading from a build that labeled clones with the email **local part**:
  existing Teleport `db` resources keep their old `dblab_user: <local part>`
  label — reconciliation matches resources by name and does not relabel them —
  so once the role matches on `{{external.email}}` those users lose access.
  Remove the affected `db` resources (or recreate the clones) so the sidecar
  re-registers them with the full-email label.

## Connecting to a clone

Once everything is running, users connect through Teleport:

```bash
# Login to Teleport
tsh login --proxy=teleport.example.com

# List available databases (clones appear automatically)
tsh db ls

# Connect to a clone
tsh db connect dblab-clone-production-<clone-id>-6000 --db-user postgres --db-name postgres

# Or use a local tunnel (works with any psql client)
tsh proxy db --tunnel dblab-clone-production-<clone-id>-6000
# Then connect to the tunnel endpoint shown in the output
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `tctl` fails with "unknown flag --proxy" | Using `--proxy` instead of `--auth-server` | Update to latest sidecar code (fixed in e25c54e5) |
| Clone registered but can't connect | No Teleport DB agent running | Start `teleport` with `db_service.enabled: true` |
| "connection refused" on IPv6 | `localhost` resolving to `[::1]` | Use `127.0.0.1` explicitly (fixed in e25c54e5) |
| TLS handshake failure | Clone doesn't have SSL enabled | Add `ssl: "on"` + cert paths to `databaseConfigs.configs` |
| "no pg_hba.conf entry" | Missing `hostssl ... cert` entry | Upgrade to DBLab Engine 4.1.0+ which includes this rule by default (see §6) |
| "password authentication failed" via Teleport | `host ... md5` rule matches before `hostssl ... cert` | Ensure `hostssl ... cert` comes before `host ... md5` in pg_hba.conf (default since DBLab Engine 4.1.0, see §6) |
| "root certificate store not available" | Missing `ssl_ca_file` | Export Teleport DB CA with `tctl auth export --type=db-client` and set `ssl_ca_file` (see §5) |
| SSL settings not applied to new clones | Snapshot created before SSL config was added | Trigger a data refresh to create a new snapshot with the updated `databaseConfigs` |
| Webhook not received | Docker networking issue | Use `host.docker.internal` or bridge IP for webhook URL |
| "access to app denied" from sidecar | Bot identity generated before role was created/updated | Regenerate the bot identity after ensuring the role exists (see §1, §2) |
| Permission denied on cert files | Wrong file ownership | `chown 999:999` on cert files |

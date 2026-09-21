# End-to-end test scripts

Each script provisions a ZFS pool, starts `dblab-server` from a built image, drives it through the
CLI, and cleans up after itself. They run on the `dle-test` shell runners in the `e2e` stage of
`engine/.gitlab-ci.yml` — on merge requests, on `master`, and on `v*` tags, which must pass the full
matrix before any publish job runs.

| Script | Retrieval mode | Run by |
|---|---|---|
| `1.synthetic.sh` | synthetic data | `.bash-test-base` |
| `2.logical_generic.sh` | logical dump/restore from a local source | `.bash-test-base` |
| `4.physical_basebackup.sh` | physical restore via `pg_basebackup` | `.bash-test-base` |
| `6.clone_upgrade.sh` | clone major-version upgrade | `.bash-test-base` |

The helpers are shared: `_prerequisites.ubuntu.sh` installs Docker, ZFS and the CLI tools,
`_zfs.file.sh` creates a file-backed pool, `_cleanup.sh` tears containers and datasets down, and
`_source_data.sh` validates the reusable source-database directory before it is bound as PGDATA.

## The gaps at 3 and 5

`3.physical_walg.sh` and `5.logical_rds.sh` were never invoked by any job. Both need infrastructure
CI does not have — a WAL-G archive with an existing base backup, and a live RDS instance with IAM
authentication — so they were replaced with manual runbooks rather than left to rot:

- [`docs/runbooks/manual-e2e-physical-walg.md`](../../docs/runbooks/manual-e2e-physical-walg.md)
- [`docs/runbooks/manual-e2e-logical-rds.md`](../../docs/runbooks/manual-e2e-logical-rds.md)

The numbering of the remaining scripts is left alone so existing pipeline logs and job output stay
searchable.

## Source data directories

`2.logical_generic.sh` and `4.physical_basebackup.sh` keep their source cluster under
`${DLE_TEST_DATA_DIR:-/var/tmp/dle_test}/<script>` and reuse it across jobs instead of reloading
pgbench every run. The default is deliberately disk-backed: on Ubuntu >= 25.04 `/tmp` is a tmpfs
sized at half of RAM, and a full PG matrix exhausts it (#781). Override `DLE_TEST_DATA_DIR` to move
the data elsewhere.

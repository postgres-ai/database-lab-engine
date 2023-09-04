import styles from './styles.module.scss'

export const tooltipText = {
  dockerTag: () => (
    <div>
      Docker image version - latest ones listed first. If unsure, pick the top
      one
    </div>
  ),
  dockerImage: () => (
    <div>
      Major PostgreSQL version (e.g., "9.6", "15"). For logical provisioning
      mode, the version used by DBLab doesn't need to match the version used on
      the source (although, it's recommended). <br />
      If you need a version that is not listed here, contact support.
    </div>
  ),
  dockerImageType: () => (
    <div>
      Docker image used to run all database containers – clones, snapshot
      preparation containers, sync containers. Although such images are based on
      traditional Docker images for Postgres, DBLab expects slightly different
      behavior: e.g., Postgres is not the first process used to start container,
      so Postgres restarts are possible, they do not trigger container state
      change. For details, see{' '}
      <a
        target={'_blank'}
        href={'https://postgres.ai/docs/database-lab/supported-databases'}
        className={styles.externalLink}
      >
        the docs
      </a>
      .
    </div>
  ),
  sharedBuffers: () => (
    <div>
      Defines default buffer pool size of each Postgres instance managed by DBLab.
      Note, that this amount of RAM is immediately allocated at Postgres startup
      time. For example, if the machine running DBLab has 32 GiB of RAM, and the
      value used here is '1GB', then the theoretical limit of the number of
      clones is 32. Practically, this limit is even lower because some memory is
      consumed by various other processes. If you need more clones, reduce the
      value of{' '}
      <span className={styles.firaCodeFont}>configs.shared_buffers</span>.
    </div>
  ),
  sharedPreloadLibraries: () => (
    <div>
      Specifies one or more shared libraries (comma-separated list) to be
      preloaded at Postgres server start (
      <a
        target={'_blank'}
        href={'https://postgresqlco.nf/doc/en/param/shared_preload_libraries/'}
        className={styles.externalLink}
      >
        details
      </a>
      ). If some libraries/extensions are missing, Postgres fails to start, so
      make sure that <span className={styles.firaCodeFont}>dockerImage</span>{' '}
      used above contains all the needed extensions.
    </div>
  ),
  host: () => (
    <div>
      Hostname/IP of database that will be used as source for data retrieval
      (full data refresh).
    </div>
  ),
  port: () => (
    <div>
      Port of database that will be used as source for data retrieval (full data
      refresh).
    </div>
  ),
  username: () => (
    <div>
      Username used to connect to database that will be used as source for data
      retrieval (full data refresh).
    </div>
  ),
  password: () => (
    <div>
      Password used to connect to database that will be used as source for data
      retrieval (full data refresh).
    </div>
  ),
  dbname: () => (
    <div>
      Database name used to connect to the source to run diagnostics queries.
      This database is not necesserarily to be copied (another field,{' '}
      <span className={styles.firaCodeFont}>databases</span>, defines which
      database to copy).
    </div>
  ),
  databases: () => (
    <div>
      Specifies list of databases Postgres server to copy at data retrieval
      (full data refresh). To specify multiple database names, provide each
      value in a separte line or use space as a divider. To copy all available
      databases, leave this value empty.
    </div>
  ),
  dumpParallelJobs: () => (
    <div>
      Number of parallel workers used to dump the source databases to disk. If
      the source is production server under load, it is not recommended to use
      more than 50% of its number of vCPUs. The higher number, the faster
      dumping is, but the higher risks of performance issues on the source
      (e.g., due to CPU or disk IO saturation).
    </div>
  ),
  pgDumpCustomOptions: () => (
    <div>
      pg_dump options to be used to create a database dump, for example:
      '--exclude-schema=repack --exclude-schema="camelStyleSchemaName"'. Note
      that due to security reasons, the current implementation supports only
      letters, numbers, hyphen, underscore, equal sign, and double quotes.
    </div>
  ),
  restoreParallelJobs: () => (
    <div>
      Number of parallel workers used to restore databases from dump to Postgres
      managed by DBLab. For initial data retrieval (very first data refresh), it
      is recommended to use the number of vCPUs available on machine running
      DBLab. With this approach, we have faster restore time, but we need to keep
      in mind that we can also have higher usage of CPU and disk IO on this
      machine (up to temporary saturation of resources). For subsequent
      refreshes, if DBLab is constantly used, it is recommended to reduce this
      value by 50% to keep some room for normal use of DBLab (such as work with
      clones).
    </div>
  ),
  pgRestoreCustomOptions: () => (
    <div>
      pg_restore options to be used to restore from a database dump, for
      example: '--exclude-schema=repack
      --exclude-schema="camelStyleSchemaName"'. Note that due to security
      reasons, the current implementation supports only letters, numbers,
      hyphen, underscore, equal sign, and double quotes.
    </div>
  ),
  timetable: () => (
    <div>
      Schedule for full data refreshes, in{' '}
      <a
        target={'_blank'}
        href={'https://en.wikipedia.org/wiki/Cron#Overview'}
        className={styles.externalLink}
      >
        crontab format
      </a>
      .
    </div>
  ),
  tuningParams: () => <div>Test</div>,
}

import styles from './styles.module.scss'

export const tooltipText = {
  dockerTag: () => (
    <div>
      Docker image version — the latest ones are listed first. If you are unsure,
      pick the first one.
    </div>
  ),
  dockerImage: () => (
    <div>
      Major PostgreSQL version (e.g., "9.6", "15"). For logical provisioning
      mode, the version used by DBLab does not need to match the version on the
      source, although matching versions is recommended. <br />
      If you need a version that is not listed here, contact support.
    </div>
  ),
  dockerImageType: () => (
    <div>
      Docker image used to run all database containers — clones, snapshot
      preparation containers, and sync containers. Although such images are based on
      traditional Docker images for PostgreSQL, DBLab expects slightly different
      behavior: for example, PostgreSQL is not the first process used to start the
      container, so PostgreSQL restarts do not trigger a container state change.
      For details, see{' '}
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
      Defines the default buffer pool size for each PostgreSQL instance managed by
      DBLab. Note that this amount of RAM is immediately allocated at PostgreSQL
      startup time. For example, if the machine running DBLab has 32 GiB of RAM and
      the value used here is '1GB', then the theoretical limit of clones is 32.
      Practically, this limit is even lower because some memory is consumed by
      other processes. If you need more clones, reduce the value of{' '}
      <span className={styles.firaCodeFont}>configs.shared_buffers</span>.
    </div>
  ),
  sharedPreloadLibraries: () => (
    <div>
      Specifies one or more shared libraries (comma-separated list) to be
      preloaded at PostgreSQL server start (
      <a
        target={'_blank'}
        href={'https://postgresqlco.nf/doc/en/param/shared_preload_libraries/'}
        className={styles.externalLink}
      >
        details
      </a>
      ). If some libraries or extensions are missing, PostgreSQL fails to start,
      so make sure that <span className={styles.firaCodeFont}>dockerImage</span>{' '}
      used above contains all required extensions.
    </div>
  ),
  host: () => (
    <div>
      Hostname or IP of the database that will be used as the source for data
      retrieval (full data refresh).
    </div>
  ),
  port: () => (
    <div>
      Port of the database that will be used as the source for data retrieval
      (full data refresh).
    </div>
  ),
  username: () => (
    <div>
      Username used to connect to the database that will be used as the source
      for data retrieval (full data refresh).
    </div>
  ),
  password: () => (
    <div>
      Password used to connect to the database that will be used as the source
      for data retrieval (full data refresh).
    </div>
  ),
  dbname: () => (
    <div>
      Database name used to connect to the source to run diagnostic queries.
      This database is not necessarily copied (another field,{' '}
      <span className={styles.firaCodeFont}>databases</span>, defines which
      database to copy).
    </div>
  ),
  databases: () => (
    <div>
      Specifies the list of databases the PostgreSQL server will copy during data
      retrieval (full data refresh). To specify multiple database names, provide
      each value on a separate line or use spaces as dividers. To copy all
      available databases, leave this value empty.
    </div>
  ),
  dumpParallelJobs: () => (
    <div>
      Number of parallel workers used to dump the source databases to disk. If
      the source is a production server under load, it is not recommended to use
      more than 50% of its available vCPUs. Increasing this number speeds up
      dumping but increases the risk of performance issues on the source (e.g.,
      due to CPU or disk I/O saturation).
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
      Number of parallel workers used to restore databases from dump to
      PostgreSQL managed by DBLab. For initial data retrieval (the first data
      refresh), it is recommended to match the number of available vCPUs on the
      machine running DBLab. This yields faster restore times but can increase
      CPU and disk I/O usage on that machine (up to temporary resource
      saturation). For subsequent refreshes, if DBLab is in continuous use, it is
      recommended to reduce this value by 50% to reserve capacity for normal
      DBLab operations (such as working with clones).
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
  tuningParams: () => (
    <div>
      Query tuning parameters. These are essential to ensure that cloned
      PostgreSQL instances generate the same plans as the source (specifically,
      they are crucial for query performance troubleshooting and optimization,
      including working with EXPLAIN plans). For details, see the{' '}
      <a
        target={'_blank'}
        href={'https://postgres.ai/docs/how-to-guides/administration/postgresql-configuration#postgresql-configuration-in-clones'}
        className={styles.externalLink}
      >
        docs
      </a>
      .
    </div>
  ),
  // Server settings
  serverHost: () => (
    <div>
      The hostname or IP address on which the DBLab API server listens.
    </div>
  ),
  serverPort: () => (
    <div>
      The port on which the DBLab API server listens.{' '}
      <strong>Changing this setting requires a restart.</strong>
    </div>
  ),
  // Provision settings
  portPoolFrom: () => (
    <div>
      The starting port number for the clone port pool. Clones will be assigned
      ports from this range.{' '}
      <strong>Changing this setting requires a restart.</strong>
    </div>
  ),
  portPoolTo: () => (
    <div>
      The ending port number for the clone port pool. Clones will be assigned
      ports from this range.{' '}
      <strong>Changing this setting requires a restart.</strong>
    </div>
  ),
  keepUserPasswords: () => (
    <div>
      When enabled, user passwords from the source database are preserved in
      clones. When disabled, all user passwords are reset during clone creation.
    </div>
  ),
  cloneAccessAddresses: () => (
    <div>
      Comma-separated list of IP addresses allowed to connect to clones. Use
      "0.0.0.0" to allow connections from any address.
    </div>
  ),
  // Cloning settings
  maxIdleMinutes: () => (
    <div>
      Maximum idle time (in minutes) before a clone is automatically deleted.
      Set to 0 to disable automatic cleanup.
    </div>
  ),
  accessHost: () => (
    <div>
      The hostname or IP address that clients should use to connect to clones.
      This is typically the external IP or hostname of the DBLab machine.
    </div>
  ),
  // Global settings
  globalDbUsername: () => (
    <div>
      Default PostgreSQL username used for database operations within DBLab.
    </div>
  ),
  globalDbName: () => (
    <div>
      Default PostgreSQL database name used for connections within DBLab.
    </div>
  ),
  // Diagnostic settings
  logsRetentionDays: () => (
    <div>
      Number of days to retain diagnostic logs. Older logs are automatically
      deleted. Set to 0 to disable automatic log cleanup.
    </div>
  ),
  // Embedded UI settings
  embeddedUIEnabled: () => (
    <div>
      Enable or disable the embedded web UI.{' '}
      <strong>Changing this setting requires a restart.</strong>
    </div>
  ),
  embeddedUIPort: () => (
    <div>
      The port on which the embedded UI listens.{' '}
      <strong>Changing this setting requires a restart.</strong>
    </div>
  ),
  // Platform settings
  platformUrl: () => (
    <div>
      URL of the Postgres.ai Platform for integration features.
    </div>
  ),
  platformProjectName: () => (
    <div>
      Project name for Postgres.ai Platform integration.
    </div>
  ),
  platformEnableTelemetry: () => (
    <div>
      Enable anonymous telemetry to help improve DBLab.
    </div>
  ),
  // Retrieval settings
  skipStartRefresh: () => (
    <div>
      When enabled, skip the initial data refresh when DBLab starts. Useful
      when you want to start DBLab without immediately connecting to the source.
    </div>
  ),
}

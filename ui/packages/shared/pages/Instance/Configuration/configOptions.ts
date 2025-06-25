export const dockerImageOptions = [
  {
    name: 'Generic PostgreSQL (postgresai/extended-postgres)',
    type: 'Generic Postgres',
  },
  { name: 'Generic PostgreSQL with PostGIS', type: 'postgis' },
  { name: 'Amazon RDS for PostgreSQL', type: 'rds' },
  { name: 'Amazon RDS Aurora for PostgreSQL', type: 'aurora' },
  { name: 'Heroku PostgreSQL', type: 'heroku' },
  { name: 'Supabase PostgreSQL', type: 'supabase' },
  { name: 'Google Cloud SQL for PostgreSQL', type: 'google-cloud-sql' },
  {
    name: 'Timescale Cloud',
    type: 'timescale-cloud',
  },
  { name: 'Custom image', type: 'custom' },
]

export const imagePgOptions = [
  {
    optionType: 'Generic Postgres',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
  {
    optionType: 'postgis',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
  {
    optionType: 'rds',
    pgDumpOptions: ['--exclude-schema=awsdms'],
    pgRestoreOptions: [],
  },
  {
    optionType: 'aurora',
    pgDumpOptions: ['--exclude-schema=awsdms'],
    pgRestoreOptions: [],
  },
  {
    optionType: 'heroku',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
  {
    optionType: 'supabase',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
  {
    optionType: 'google-cloud-sql',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
  {
    optionType: 'timescale-cloud',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
]

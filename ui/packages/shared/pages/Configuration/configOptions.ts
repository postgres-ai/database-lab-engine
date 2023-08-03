export const dockerImageOptions = [
  {
    name: 'Generic Postgres (postgresai/extended-postgres)',
    type: 'Generic Postgres',
  },
  { name: 'Generic Postgres with PostGIS', type: 'postgis' },
  { name: 'Amazon RDS for Postgres', type: 'rds' },
  { name: 'Amazon RDS Aurora for Postgres', type: 'aurora' },
  { name: 'Heroku Postgres', type: 'heroku' },
  { name: 'Supabase Postgres', type: 'supabase' },
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

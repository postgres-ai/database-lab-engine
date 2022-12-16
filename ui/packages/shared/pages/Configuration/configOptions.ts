export const dockerImageOptions = [
  {
    name: 'Generic Postgres (postgresai/extended-postgres)',
    type: 'Generic Postgres',
  },
  { name: 'Generic Postgres with PostGIS', type: 'postgis' },
  { name: 'Amazon RDS for Postgres', type: 'rds' },
  { name: 'Amazon RDS Aurora for Postgres', type: 'rdsAurora' },
  { name: 'Heroku Postgres', type: 'heroku' },
  { name: 'Supabase Postgres', type: 'supabase' },
  { name: 'Custom image', type: 'custom' },
]

export const defaultPgDumpOptions = [
  {
    optionType: 'Generic Postgres',
    addDefaultOptions: [],
  },
  {
    optionType: 'postgis',
    addDefaultOptions: [],
  },
  {
    optionType: 'rds',
    addDefaultOptions: [],
  },
  {
    optionType: 'rdsAurora',
    addDefaultOptions: [],
  },
  {
    optionType: 'heroku',
    addDefaultOptions: [],
  },
  {
    optionType: 'supabase',
    addDefaultOptions: [],
  },
]

export const defaultPgRestoreOptions = [
  {
    optionType: 'Generic Postgres',
    addDefaultOptions: [],
  },
  {
    optionType: 'postgis',
    addDefaultOptions: [],
  },
  {
    optionType: 'rds',
    addDefaultOptions: ['---exclude-schema=awsdms'],
  },
  {
    optionType: 'rdsAurora',
    addDefaultOptions: ['---exclude-schema=awsdms'],
  },
  {
    optionType: 'heroku',
    addDefaultOptions: [],
  },
  {
    optionType: 'supabase',
    addDefaultOptions: [],
  },
]

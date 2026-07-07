import { describe, it, expect, vi, beforeEach } from 'vitest'

const requestMock = vi.fn()

vi.mock('helpers/request', () => ({
  request: (...args: unknown[]) => requestMock(...args),
}))

import { updateConfig } from './updateConfig'
import type { Config } from '@postgres.ai/shared/types/api/entities/config'

const okResponse = () => new Response(null, { status: 200 })

const baseConfig = (overrides: Partial<Config> = {}): Config =>
  ({
    debug: false,
    retrievalMode: 'logical',
    physicalTool: '',
    physicalDockerImage: '',
    physicalSyncEnabled: false,
    physicalWalgBackupName: '',
    physicalPgbackrestStanza: '',
    physicalPgbackrestDelta: false,
    physicalEnvs: [],
    dockerImage: 'postgres-image',
    dockerPath: 'postgresai/extended-postgres:18',
    tuningParams: '',
    sharedBuffers: '1GB',
    sharedPreloadLibraries: 'pg_stat_statements',
    timetable: '0 0 * * *',
    dbname: 'shop',
    host: 'db.example.com',
    port: '5432',
    username: 'app',
    password: 'pw',
    databases: 'shop',
    dumpParallelJobs: '4',
    dumpIgnoreErrors: false,
    restoreParallelJobs: '4',
    restoreIgnoreErrors: false,
    restoreConfigs: '',
    pgDumpCustomOptions: '',
    pgRestoreCustomOptions: '',
    ...overrides,
  } as Config)

const lastBody = () => {
  const args = requestMock.mock.lastCall as [string, { body: string }]
  return JSON.parse(args[1].body)
}

describe('updateConfig — mode-aware payload', () => {
  beforeEach(() => {
    requestMock.mockReset()
    requestMock.mockResolvedValue(okResponse())
  })

  it('logical mode writes logicalDump + logicalRestore and no physicalRestore', async () => {
    await updateConfig(baseConfig({ retrievalMode: 'logical' }))

    const body = lastBody()
    expect(body.retrievalMode).toBe('logical')
    expect(body.retrieval.spec.logicalDump.options.source.connection).toMatchObject({
      host: 'db.example.com',
      dbname: 'shop',
      username: 'app',
      port: '5432',
    })
    expect(body.retrieval.spec.logicalRestore).toBeDefined()
    expect(body.retrieval.spec.physicalRestore).toBeUndefined()
  })

  it('logical mode clears source.connectionString so a stale value cannot shadow discrete edits', async () => {
    await updateConfig(baseConfig({ retrievalMode: 'logical' }))

    const source = lastBody().retrieval.spec.logicalDump.options.source
    expect(source.connectionString).toBe('')
    expect(source.connection).toMatchObject({ host: 'db.example.com', dbname: 'shop' })
  })

  it('physical mode does not send a logical connectionString field', async () => {
    await updateConfig(baseConfig({ retrievalMode: 'physical', physicalTool: 'walg' }))

    const body = lastBody()
    expect(body.retrieval.spec.logicalDump).toBeUndefined()
    expect(JSON.stringify(body)).not.toContain('connectionString')
  })

  it('physical+walg mode writes walg + envs and omits logical blocks', async () => {
    await updateConfig(
      baseConfig({
        retrievalMode: 'physical',
        physicalTool: 'walg',
        physicalWalgBackupName: 'LATEST',
        physicalDockerImage: 'postgresai/extended-postgres:18-0.6.2',
        physicalSyncEnabled: true,
        physicalEnvs: [
          { key: 'WALG_S3_PREFIX', value: 's3://bucket/prefix' },
          { key: 'AWS_REGION', value: 'eu-west-1' },
        ],
      }),
    )

    const body = lastBody()
    expect(body.retrievalMode).toBe('physical')
    expect(body.retrieval.spec.logicalDump).toBeUndefined()
    expect(body.retrieval.spec.logicalRestore).toBeUndefined()

    const opts = body.retrieval.spec.physicalRestore.options
    expect(opts.tool).toBe('walg')
    expect(opts.walg).toEqual({ backupName: 'LATEST' })
    expect(opts.pgbackrest).toBeUndefined()
    expect(opts.dockerImage).toBe('postgresai/extended-postgres:18-0.6.2')
    expect(opts.sync).toEqual({ enabled: true })
    expect(opts.envs).toEqual({
      WALG_S3_PREFIX: 's3://bucket/prefix',
      AWS_REGION: 'eu-west-1',
    })
  })

  it('physical+pgbackrest mode writes pgbackrest stanza/delta and skips walg block', async () => {
    await updateConfig(
      baseConfig({
        retrievalMode: 'physical',
        physicalTool: 'pgbackrest',
        physicalPgbackrestStanza: 'main',
        physicalPgbackrestDelta: true,
        physicalEnvs: [
          { key: 'PGBACKREST_REPO1_S3_BUCKET', value: 'my-backup-bucket' },
        ],
      }),
    )

    const opts = lastBody().retrieval.spec.physicalRestore.options
    expect(opts.tool).toBe('pgbackrest')
    expect(opts.pgbackrest).toEqual({ stanza: 'main', delta: true })
    expect(opts.walg).toBeUndefined()
    expect(opts.envs).toEqual({
      PGBACKREST_REPO1_S3_BUCKET: 'my-backup-bucket',
    })
  })

  it('physical mode drops empty envs rows so the YAML is not polluted', async () => {
    await updateConfig(
      baseConfig({
        retrievalMode: 'physical',
        physicalTool: 'walg',
        physicalEnvs: [
          { key: 'WALG_S3_PREFIX', value: 's3://bucket' },
          { key: '', value: 'ignored value' },
          { key: '', value: '' },
        ],
      }),
    )

    const opts = lastBody().retrieval.spec.physicalRestore.options
    expect(opts.envs).toEqual({ WALG_S3_PREFIX: 's3://bucket' })
  })

  it('logical mode preserves the empty-port omission rule from useForm', async () => {
    await updateConfig(baseConfig({ retrievalMode: 'logical', port: '' }))
    const body = lastBody()
    expect(body.retrieval.spec.logicalDump.options.source.connection.port).toBeUndefined()
  })

  it('logical mode omits empty parallelJobs so the engine does not parse "" as int', async () => {
    await updateConfig(
      baseConfig({ retrievalMode: 'logical', dumpParallelJobs: '', restoreParallelJobs: '' }),
    )
    const body = lastBody()
    expect(body.retrieval.spec.logicalDump.options.parallelJobs).toBeUndefined()
    expect(body.retrieval.spec.logicalRestore.options.parallelJobs).toBeUndefined()
  })

  it('object-shaped tuningParams (SimpleMode projection) lands in databaseConfigs.configs as real keys', async () => {
    const tuning = { work_mem: '8MB', random_page_cost: '1.1' } as unknown as string
    await updateConfig(baseConfig({ tuningParams: tuning }))
    const configs = lastBody().databaseConfigs.configs
    expect(configs).toEqual(
      expect.objectContaining({
        shared_buffers: '1GB',
        shared_preload_libraries: 'pg_stat_statements',
        work_mem: '8MB',
        random_page_cost: '1.1',
      }),
    )
    expect(configs['0']).toBeUndefined()
  })
})

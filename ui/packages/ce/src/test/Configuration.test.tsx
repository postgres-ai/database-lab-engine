import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

import { StoresProvider } from '@postgres.ai/shared/pages/Instance/context'
import { Configuration } from '@postgres.ai/shared/pages/Instance/Configuration'
import { Config } from '@postgres.ai/shared/types/api/entities/config'

const physicalWalgConfig = {
  retrievalMode: 'physical',
  physicalTool: 'walg',
  physicalDockerImage: 'postgresai/extended-postgres:15',
  physicalSyncEnabled: false,
  physicalWalgBackupName: 'LATEST',
  physicalPgbackrestStanza: '',
  physicalPgbackrestDelta: false,
  physicalEnvs: [{ key: 'WALG_S3_PREFIX', value: 's3://bucket/path' }],
  dockerImage: 'postgresai/extended-postgres:15',
  dockerImageType: 'Generic Postgres',
  dockerTag: '15',
  dockerPath: 'postgresai/extended-postgres:15',
  tuningParams: '',
  sharedBuffers: '',
  sharedPreloadLibraries: '',
  debug: false,
  timetable: '',
  dbname: '',
  host: '',
  port: '',
  username: '',
  password: '',
  databases: '',
  dumpParallelJobs: '',
  dumpIgnoreErrors: false,
  restoreParallelJobs: '',
  restoreIgnoreErrors: false,
  restoreConfigs: '',
  pgDumpCustomOptions: '',
  pgRestoreCustomOptions: '',
} as unknown as Config

const buildStores = (overrides: Record<string, unknown> = {}) => ({
  main: {
    config: physicalWalgConfig,
    isConfigurationLoading: false,
    fullConfig: undefined,
    configError: null,
    getFullConfigError: null,
    updateConfig: vi.fn().mockResolvedValue({}),
    getFullConfig: vi.fn().mockResolvedValue(undefined),
    getSeImages: vi.fn().mockResolvedValue(undefined),
    testDbSource: vi.fn().mockResolvedValue({ response: null, error: null }),
    getEngine: vi.fn().mockResolvedValue({ edition: 'community' }),
    probeSource: vi.fn().mockResolvedValue({ response: null, error: null }),
    ...overrides,
  },
  clonesModal: {},
  snapshotsModal: {},
})

const renderConfig = (storeOverrides: Record<string, unknown> = {}) => {
  const stores = buildStores(storeOverrides)
  const utils = render(
    // @ts-expect-error — partial Stores is fine in tests
    <StoresProvider value={stores}>
      <Configuration
        instanceId="inst-1"
        switchActiveTab={vi.fn()}
        reload={vi.fn()}
      />
    </StoresProvider>,
  )
  return { ...utils, stores }
}

describe('Configuration — physical-mode gating', () => {
  it('renders the WAL-G backup-name input as enabled when retrieval mode is physical', async () => {
    renderConfig()

    const input = await waitFor(() =>
      screen.getByTestId('walg-backup-name') as HTMLInputElement,
    )
    expect(input).not.toBeDisabled()
  })

  it('does not render the legacy logical-only snackbar in physical mode', async () => {
    renderConfig()

    await waitFor(() => expect(screen.getByTestId('walg-backup-name')).toBeInTheDocument())
    expect(
      screen.queryByText(/Configuration editing is only available in logical mode/i),
    ).not.toBeInTheDocument()
  })

  it('does not render the legacy logical-only snackbar in logical mode either', async () => {
    const logicalConfig = {
      ...physicalWalgConfig,
      retrievalMode: 'logical',
      host: 'db.example.com',
      port: '5432',
      username: 'app',
      dbname: 'shop',
    } as unknown as Config
    renderConfig({ config: logicalConfig })

    expect(
      screen.queryByText(/Configuration editing is only available in logical mode/i),
    ).not.toBeInTheDocument()
  })
})

const headerLabel = () =>
  screen.getByText(/Data retrieval mode/i).closest('p')?.textContent ?? ''

describe('Configuration — lazy mode-default seeding', () => {
  const tabSelected = (name: 'Simple' | 'Expert') =>
    screen.getByRole('tab', { name }).getAttribute('aria-selected') === 'true'

  it('falls back to Simple while config is loading (configData === null)', async () => {
    renderConfig({ config: null })

    await waitFor(() => expect(screen.getByRole('tab', { name: 'Simple' })).toBeInTheDocument())
    expect(tabSelected('Simple')).toBe(true)
    expect(tabSelected('Expert')).toBe(false)
  })

  it('seeds Expert once config arrives with a host', async () => {
    const logicalConfig = {
      ...physicalWalgConfig,
      retrievalMode: 'logical',
      host: 'db.example.com',
      port: '5432',
      username: 'app',
      dbname: 'shop',
    } as unknown as Config
    renderConfig({ config: logicalConfig })

    await waitFor(() => expect(tabSelected('Expert')).toBe(true))
  })

  it('seeds Expert once config arrives in physical mode (no host field)', async () => {
    renderConfig()

    await waitFor(() => expect(tabSelected('Expert')).toBe(true))
  })

  it('locks initialMode after first non-null configData — refetch does not flip the tab', async () => {
    const initial = {
      ...physicalWalgConfig,
      retrievalMode: 'logical',
      host: 'db.example.com',
      port: '5432',
      username: 'app',
      dbname: 'shop',
    } as unknown as Config
    const { rerender, stores } = renderConfig({ config: initial })

    await waitFor(() => expect(tabSelected('Expert')).toBe(true))

    const refreshed = { ...(initial as object) } as unknown as Config
    stores.main.config = refreshed
    rerender(
      // @ts-expect-error — partial Stores is fine in tests
      <StoresProvider value={stores}>
        <Configuration
          instanceId="inst-1"
          switchActiveTab={vi.fn()}
          reload={vi.fn()}
        />
      </StoresProvider>,
    )

    expect(tabSelected('Expert')).toBe(true)
  })

  it('user-picked tab wins over the seeded default', async () => {
    renderConfig()

    await waitFor(() => expect(tabSelected('Expert')).toBe(true))

    fireEvent.click(screen.getByRole('tab', { name: 'Simple' }))

    expect(tabSelected('Simple')).toBe(true)
    expect(tabSelected('Expert')).toBe(false)
  })
})

const logicalProposedSample = {
  source: { host: 'db.example.com', port: 5432, username: 'app', dbname: 'shop' },
  detectedProvider: 'rds',
  dockerImage: 'rds',
  dockerTag: '',
  pgMajorVersion: 15,
  databases: ['shop'],
  sharedBuffers: '4GB',
  memoryProbed: true,
  sharedPreloadLibraries: 'pg_stat_statements',
  queryTuning: { work_mem: '8MB', random_page_cost: '1.1' },
}

describe('Configuration — Simple → Edit → Apply', () => {
  it('applies from Expert after Simple→Edit without throwing on object-shaped tuningParams', async () => {
    const updateConfig = vi.fn().mockResolvedValue({})
    const probeSource = vi
      .fn()
      .mockResolvedValue({ response: logicalProposedSample, error: null })
    renderConfig({ config: null, updateConfig, probeSource })

    fireEvent.change(screen.getByTestId('simple-url'), {
      target: { value: 'postgres://db.example.com/shop' },
    })
    fireEvent.change(screen.getByTestId('simple-password'), {
      target: { value: 'secret' },
    })
    fireEvent.click(screen.getByTestId('simple-detect'))

    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())

    fireEvent.click(screen.getByTestId('preview-edit'))

    const applyButton = await screen.findByText('Apply changes')
    fireEvent.click(applyButton)

    await waitFor(() => expect(updateConfig).toHaveBeenCalledTimes(1))
    const submitted = updateConfig.mock.calls[0][0]
    expect(submitted.tuningParams).toEqual({
      work_mem: '8MB',
      random_page_cost: '1.1',
    })
  })
})

describe('Configuration — Header retrievalMode label', () => {
  it('renders "physical" in the Header when the config is physical', async () => {
    renderConfig()

    await waitFor(() => expect(headerLabel()).toMatch(/physical/i))
  })

  it('renders "logical" in the Header when the config is logical', async () => {
    const logicalConfig = {
      ...physicalWalgConfig,
      retrievalMode: 'logical',
      host: 'db.example.com',
      port: '5432',
      username: 'app',
      dbname: 'shop',
    } as unknown as Config
    renderConfig({ config: logicalConfig })

    await waitFor(() => expect(headerLabel()).toMatch(/logical/i))
  })

  it('updates the Header label when the retrieval-mode radio is switched', async () => {
    renderConfig()

    await waitFor(() => expect(headerLabel()).toMatch(/physical/i))

    const logicalRadio = screen.getByLabelText('Logical (dump/restore)')
    fireEvent.click(logicalRadio)

    await waitFor(() => expect(headerLabel()).toMatch(/logical/i))
  })
})

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

import { StoresProvider } from '@postgres.ai/shared/pages/Instance/context'
import {
  SimpleMode,
  buildProjectionFromProposed,
  resolveProbeImage,
} from '@postgres.ai/shared/pages/Instance/Configuration/SimpleMode'
import { SeImages } from '@postgres.ai/shared/types/api/endpoints/getSeImages'
import { ProposedConfig } from '@postgres.ai/shared/types/api/endpoints/probeSource'

const proposedSample: ProposedConfig = {
  source: { host: 'db.example.com', port: 5432, username: 'app', dbname: 'shop' },
  detectedProvider: 'rds',
  dockerImage: 'rds',
  dockerTag: '',
  pgMajorVersion: 15,
  databases: ['shop'],
  sharedBuffers: '4GB',
  memoryProbed: true,
  sharedPreloadLibraries: 'pg_stat_statements,pgaudit',
  queryTuning: { work_mem: '8MB', random_page_cost: '1.1' },
}

const seImageSample: SeImages[] = [
  {
    package_group: 'rds',
    pg_major_version: '15',
    tag: '15-se-1',
    location: 'registry.example.com/se/rds:15-se-1',
    pg_config_presets: {
      shared_preload_libraries: 'pg_stat_statements,pg_cron,pgaudit',
    },
  },
]

const buildStores = (
  overrides: {
    probeSource?: unknown
    updateConfig?: unknown
    configError?: string | null
    getSeImages?: unknown
    fullRefresh?: unknown
  } = {},
) => ({
  main: {
    probeSource:
      overrides.probeSource ??
      vi.fn().mockResolvedValue({ response: proposedSample, error: null }),
    updateConfig: overrides.updateConfig ?? vi.fn().mockResolvedValue({}),
    configError: overrides.configError ?? null,
    getSeImages: overrides.getSeImages ?? vi.fn().mockResolvedValue(undefined),
    fullRefresh:
      overrides.fullRefresh ??
      vi.fn().mockResolvedValue({ response: 'Full refresh started', error: null }),
  },
  clonesModal: {},
  snapshotsModal: {},
})

const renderMode = (props: Partial<Parameters<typeof SimpleMode>[0]> = {}, storeOverrides = {}) => {
  const stores = buildStores(storeOverrides)
  const utils = render(
    // @ts-expect-error — partial Stores is fine in tests
    <StoresProvider value={stores}>
      <SimpleMode instanceId="inst-1" {...props} />
    </StoresProvider>,
  )
  return { ...utils, stores }
}

describe('SimpleMode form', () => {
  beforeEach(() => vi.clearAllMocks())

  it('detect button is disabled until URL and password are filled', () => {
    renderMode()
    const button = screen.getByTestId('simple-detect') as HTMLButtonElement
    expect(button.disabled).toBe(true)

    fireEvent.change(screen.getByTestId('simple-url'), {
      target: { value: 'postgres://db.example.com/shop' },
    })
    expect((screen.getByTestId('simple-detect') as HTMLButtonElement).disabled).toBe(true)

    fireEvent.change(screen.getByTestId('simple-password'), {
      target: { value: 'secret' },
    })
    expect((screen.getByTestId('simple-detect') as HTMLButtonElement).disabled).toBe(false)
  })

  it('clicking Detect calls probeSource with URL and password and renders the preview', async () => {
    const { stores } = renderMode()

    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'postgres://db.example.com/shop' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByTestId('simple-detect'))

    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    expect(stores.main.probeSource).toHaveBeenCalledWith({
      url: 'postgres://db.example.com/shop',
      password: 'secret',
    })
  })

  it('renders an inline error when probeSource fails', async () => {
    const probeFail = vi.fn().mockResolvedValue({
      response: null,
      error: { status: 400, message: 'bad url' },
    })
    renderMode({}, { probeSource: probeFail })

    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'x' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'y' } })
    fireEvent.click(screen.getByTestId('simple-detect'))

    await waitFor(() => expect(screen.getByTestId('probe-error')).toHaveTextContent('bad url'))
    expect(screen.queryByTestId('preview-card')).not.toBeInTheDocument()
  })

  it('clicking Apply calls updateConfig with the mapped projection', async () => {
    const updateConfig = vi.fn().mockResolvedValue({})
    renderMode({}, { updateConfig })

    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'postgres://db.example.com/shop' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByTestId('simple-detect'))

    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    fireEvent.click(screen.getByTestId('preview-apply'))

    await waitFor(() => expect(updateConfig).toHaveBeenCalledTimes(1))
    const [projection, instanceId] = updateConfig.mock.calls[0]
    expect(instanceId).toBe('inst-1')
    expect(projection).toMatchObject({
      host: 'db.example.com',
      port: '5432',
      username: 'app',
      dbname: 'shop',
      password: 'secret',
      sharedBuffers: '4GB',
      sharedPreloadLibraries: 'pg_stat_statements,pgaudit',
      dockerImageType: 'rds',
    })
    expect(projection.tuningParams).toEqual({ work_mem: '8MB', random_page_cost: '1.1' })
    expect(typeof projection.tuningParams).toBe('object')
    expect(projection.tuningParams).toEqual(expect.objectContaining({ work_mem: '8MB' }))
  })

  it('clicking Apply invokes onApplied on success', async () => {
    const onApplied = vi.fn()
    renderMode({ onApplied })

    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'x' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'y' } })
    fireEvent.click(screen.getByTestId('simple-detect'))
    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    fireEvent.click(screen.getByTestId('preview-apply'))
    await waitFor(() => expect(onApplied).toHaveBeenCalledTimes(1))
  })

  it('clicking Apply triggers a full refresh after the config is applied', async () => {
    const fullRefresh = vi.fn().mockResolvedValue({ response: 'Full refresh started', error: null })
    renderMode({}, { fullRefresh })

    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'x' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'y' } })
    fireEvent.click(screen.getByTestId('simple-detect'))
    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    fireEvent.click(screen.getByTestId('preview-apply'))
    await waitFor(() => expect(fullRefresh).toHaveBeenCalledWith('inst-1'))
  })

  it('renders Apply error when the full refresh fails', async () => {
    const fullRefresh = vi.fn().mockResolvedValue({ response: null, error: { message: 'pool is busy' } })
    const onApplied = vi.fn()
    renderMode({ onApplied }, { fullRefresh })

    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'x' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'y' } })
    fireEvent.click(screen.getByTestId('simple-detect'))
    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    fireEvent.click(screen.getByTestId('preview-apply'))
    await waitFor(() => expect(screen.getByTestId('apply-error')).toHaveTextContent('pool is busy'))
    expect(onApplied).not.toHaveBeenCalled()
  })

  it('renders Apply error when a request rejects instead of leaving the button stuck', async () => {
    const updateConfig = vi.fn().mockRejectedValue(new Error('network down'))
    const onApplied = vi.fn()
    renderMode({ onApplied }, { updateConfig })

    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'x' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'y' } })
    fireEvent.click(screen.getByTestId('simple-detect'))
    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    fireEvent.click(screen.getByTestId('preview-apply'))
    await waitFor(() => expect(screen.getByTestId('apply-error')).toHaveTextContent('network down'))
    expect(onApplied).not.toHaveBeenCalled()
  })

  it('renders Apply error when updateConfig returns null', async () => {
    const updateConfig = vi.fn().mockResolvedValue(undefined)
    renderMode({}, { updateConfig, configError: 'engine refused config' })

    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'x' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'y' } })
    fireEvent.click(screen.getByTestId('simple-detect'))
    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    fireEvent.click(screen.getByTestId('preview-apply'))
    await waitFor(() => expect(screen.getByTestId('apply-error')).toHaveTextContent('engine refused config'))
  })

  it('clicking Edit-before-applying invokes onEdit with proposed config and password', async () => {
    const onEdit = vi.fn()
    renderMode({ onEdit })

    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'x' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByTestId('simple-detect'))
    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    fireEvent.click(screen.getByTestId('preview-edit'))
    expect(onEdit).toHaveBeenCalledWith(proposedSample, 'secret')
  })

  it('renders the generic-provider callout when detection falls back', async () => {
    const probe = vi.fn().mockResolvedValue({
      response: { ...proposedSample, detectedProvider: 'generic', dockerImage: 'generic' },
      error: null,
    })
    renderMode({}, { probeSource: probe })
    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'x' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'y' } })
    fireEvent.click(screen.getByTestId('simple-detect'))
    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    expect(screen.getByText(/Could not detect a managed cloud provider/i)).toBeInTheDocument()
  })

  it('renders the memory-fallback callout when memoryProbed is false', async () => {
    const probe = vi.fn().mockResolvedValue({
      response: { ...proposedSample, memoryProbed: false, sharedBuffers: '1GB' },
      error: null,
    })
    renderMode({}, { probeSource: probe })
    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'x' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'y' } })
    fireEvent.click(screen.getByTestId('simple-detect'))
    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    expect(screen.getByText(/Could not detect host memory/i)).toBeInTheDocument()
  })

  it('ships the platform SE image and its preset when the SE catalog is available', async () => {
    const updateConfig = vi.fn().mockResolvedValue({})
    const getSeImages = vi.fn().mockResolvedValue(seImageSample)
    renderMode({}, { updateConfig, getSeImages })

    fireEvent.change(screen.getByTestId('simple-url'), { target: { value: 'x' } })
    fireEvent.change(screen.getByTestId('simple-password'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByTestId('simple-detect'))

    await waitFor(() => expect(screen.getByTestId('preview-card')).toBeInTheDocument())
    expect(getSeImages).toHaveBeenCalledWith({ packageGroup: 'rds' })
    expect(screen.getByText('registry.example.com/se/rds:15-se-1')).toBeInTheDocument()

    fireEvent.click(screen.getByTestId('preview-apply'))
    await waitFor(() => expect(updateConfig).toHaveBeenCalledTimes(1))
    expect(updateConfig.mock.calls[0][0]).toMatchObject({
      dockerPath: 'registry.example.com/se/rds:15-se-1',
      dockerTag: '15-se-1',
      sharedPreloadLibraries: 'pg_stat_statements,pg_cron,pgaudit',
    })
  })
})

describe('buildProjectionFromProposed', () => {
  it('produces a generic-image projection when provider is generic', () => {
    const out = buildProjectionFromProposed(
      { ...proposedSample, detectedProvider: 'generic', dockerImage: 'generic' },
      'pw',
    )
    expect(out).toMatchObject({
      dockerImageType: 'Generic Postgres',
      dockerTag: '15-0.5.3',
      dockerPath: 'postgresai/extended-postgres:15-0.5.3',
      dockerImage: '15',
      host: 'db.example.com',
      port: '5432',
      username: 'app',
      dbname: 'shop',
      password: 'pw',
      databases: 'shop',
      sharedBuffers: '4GB',
      sharedPreloadLibraries: 'pg_stat_statements,pgaudit',
    })
  })

  it('falls back to the generic image for managed providers CE cannot pull', () => {
    const out = buildProjectionFromProposed(proposedSample, 'pw')
    expect(out).toMatchObject({
      dockerImageType: 'rds',
      dockerTag: '',
      dockerPath: 'postgresai/extended-postgres:15-0.5.3',
      dockerImage: 'rds',
    })
  })

  it('applies a resolved SE image and preset over the generic fallback', () => {
    const out = buildProjectionFromProposed(proposedSample, 'pw', {
      dockerPath: 'registry.example.com/se/rds:15-se-1',
      dockerTag: '15-se-1',
      sharedPreloadLibraries: 'pg_cron,pgaudit',
      isSe: true,
    })
    expect(out).toMatchObject({
      dockerImageType: 'rds',
      dockerPath: 'registry.example.com/se/rds:15-se-1',
      dockerTag: '15-se-1',
      sharedPreloadLibraries: 'pg_cron,pgaudit',
    })
  })

  it('returns tuningParams as an object so updateConfig can spread real keys', () => {
    const out = buildProjectionFromProposed(proposedSample, 'pw')
    expect(typeof out.tuningParams).toBe('object')
    expect(out.tuningParams).toEqual({ work_mem: '8MB', random_page_cost: '1.1' })
    expect(out.tuningParams).not.toBe('work_mem=8MB\nrandom_page_cost=1.1')
  })

  it('produces an empty tuningParams object when queryTuning is empty', () => {
    const out = buildProjectionFromProposed({ ...proposedSample, queryTuning: {} }, 'pw')
    expect(out.tuningParams).toEqual({})
  })
})

describe('resolveProbeImage', () => {
  it('returns the SE image and preset when the catalog has the major version', async () => {
    const getSeImages = vi.fn().mockResolvedValue(seImageSample)
    const out = await resolveProbeImage(proposedSample, getSeImages)
    expect(getSeImages).toHaveBeenCalledWith({ packageGroup: 'rds' })
    expect(out).toEqual({
      dockerPath: 'registry.example.com/se/rds:15-se-1',
      dockerTag: '15-se-1',
      sharedPreloadLibraries: 'pg_stat_statements,pg_cron,pgaudit',
      isSe: true,
    })
  })

  it('falls back to the generic image when the SE catalog is empty (CE)', async () => {
    const getSeImages = vi.fn().mockResolvedValue(undefined)
    const out = await resolveProbeImage(proposedSample, getSeImages)
    expect(out).toMatchObject({
      dockerPath: 'postgresai/extended-postgres:15-0.5.3',
      sharedPreloadLibraries: 'pg_stat_statements,pgaudit',
      isSe: false,
    })
  })

  it('falls back to generic when the SE catalog lacks the detected major version', async () => {
    const getSeImages = vi.fn().mockResolvedValue([{ ...seImageSample[0], pg_major_version: '14' }])
    const out = await resolveProbeImage(proposedSample, getSeImages)
    expect(out.isSe).toBe(false)
    expect(out.dockerPath).toBe('postgresai/extended-postgres:15-0.5.3')
  })

  it('does not query the SE catalog for a generic provider', async () => {
    const getSeImages = vi.fn().mockResolvedValue(seImageSample)
    const out = await resolveProbeImage(
      { ...proposedSample, detectedProvider: 'generic', dockerImage: 'generic' },
      getSeImages,
    )
    expect(getSeImages).not.toHaveBeenCalled()
    expect(out.isSe).toBe(false)
  })
})

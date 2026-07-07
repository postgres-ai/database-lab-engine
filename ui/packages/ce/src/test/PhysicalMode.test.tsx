import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

import { PhysicalMode } from '@postgres.ai/shared/pages/Instance/Configuration/PhysicalMode'
import { FormValues } from '@postgres.ai/shared/pages/Instance/Configuration/useForm'

const baseValues = (): FormValues => ({
  debug: false,
  dockerImage: '',
  dockerTag: '',
  dockerPath: '',
  dockerImageType: '',
  sharedBuffers: '',
  sharedPreloadLibraries: '',
  tuningParams: '',
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
  retrievalMode: 'physical',
  physicalTool: '',
  physicalDockerImage: '',
  physicalSyncEnabled: false,
  physicalWalgBackupName: '',
  physicalPgbackrestStanza: '',
  physicalPgbackrestDelta: false,
  physicalEnvs: [],
})

describe('PhysicalMode', () => {
  it('renders the tool selector and no sub-form until a tool is picked', () => {
    const onChange = vi.fn()
    render(<PhysicalMode values={baseValues()} onChange={onChange} />)

    expect(screen.getByLabelText('physical restore tool')).toBeInTheDocument()
    expect(screen.queryByTestId('walg-form')).not.toBeInTheDocument()
    expect(screen.queryByTestId('pgbackrest-form')).not.toBeInTheDocument()
    expect(screen.queryByTestId('physical-sync')).not.toBeInTheDocument()
  })

  it('renders the WAL-G sub-form and Sync section when tool is walg', () => {
    const values = { ...baseValues(), physicalTool: 'walg' as const }
    render(<PhysicalMode values={values} onChange={vi.fn()} />)

    expect(screen.getByTestId('walg-form')).toBeInTheDocument()
    expect(screen.queryByTestId('pgbackrest-form')).not.toBeInTheDocument()
    expect(screen.getByTestId('physical-sync')).toBeInTheDocument()
  })

  it('renders the pgBackRest sub-form and Sync section when tool is pgbackrest', () => {
    const values = { ...baseValues(), physicalTool: 'pgbackrest' as const }
    render(<PhysicalMode values={values} onChange={vi.fn()} />)

    expect(screen.getByTestId('pgbackrest-form')).toBeInTheDocument()
    expect(screen.queryByTestId('walg-form')).not.toBeInTheDocument()
    expect(screen.getByTestId('physical-sync')).toBeInTheDocument()
  })

  it('shows the customTool banner and hides the radio when tool is customTool', () => {
    const values = { ...baseValues(), physicalTool: 'customTool' as const }
    render(<PhysicalMode values={values} onChange={vi.fn()} />)

    expect(screen.getByTestId('physical-custom-tool-banner')).toBeInTheDocument()
    expect(screen.queryByLabelText('physical restore tool')).not.toBeInTheDocument()
    expect(screen.queryByTestId('walg-form')).not.toBeInTheDocument()
    expect(screen.queryByTestId('pgbackrest-form')).not.toBeInTheDocument()
  })

  it('calls onChange when a different tool is selected', () => {
    const onChange = vi.fn()
    const values = { ...baseValues(), physicalTool: 'walg' as const }
    render(<PhysicalMode values={values} onChange={onChange} />)

    const pgbackrestRadio = screen.getByLabelText('pgBackRest')
    fireEvent.click(pgbackrestRadio)

    expect(onChange).toHaveBeenCalledWith('physicalTool', 'pgbackrest')
  })

  it('WAL-G sub-form binds BackupName to physicalWalgBackupName', () => {
    const onChange = vi.fn()
    const values = {
      ...baseValues(),
      physicalTool: 'walg' as const,
      physicalWalgBackupName: 'LATEST',
    }
    render(<PhysicalMode values={values} onChange={onChange} />)

    const input = screen.getByTestId('walg-backup-name') as HTMLInputElement
    expect(input.value).toBe('LATEST')

    fireEvent.change(input, { target: { value: 'backup_20260518' } })
    expect(onChange).toHaveBeenCalledWith('physicalWalgBackupName', 'backup_20260518')
  })

  it('pgBackRest sub-form binds Stanza and Delta', () => {
    const onChange = vi.fn()
    const values = {
      ...baseValues(),
      physicalTool: 'pgbackrest' as const,
      physicalPgbackrestStanza: 'main',
      physicalPgbackrestDelta: false,
    }
    render(<PhysicalMode values={values} onChange={onChange} />)

    const stanza = screen.getByTestId('pgbackrest-stanza') as HTMLInputElement
    expect(stanza.value).toBe('main')

    fireEvent.change(stanza, { target: { value: 'replica' } })
    expect(onChange).toHaveBeenCalledWith('physicalPgbackrestStanza', 'replica')

    const delta = screen.getByTestId('pgbackrest-delta') as HTMLInputElement
    fireEvent.click(delta)
    expect(onChange).toHaveBeenCalledWith('physicalPgbackrestDelta', true)
  })

  it('envs editor adds and removes rows', () => {
    const onChange = vi.fn()
    const values = { ...baseValues(), physicalTool: 'walg' as const }
    render(<PhysicalMode values={values} onChange={onChange} />)

    fireEvent.click(screen.getByTestId('envs-add'))
    expect(onChange).toHaveBeenLastCalledWith('physicalEnvs', [
      { key: '', value: '' },
    ])
  })

  it('envs editor inserts a suggestion via click-to-add', () => {
    const onChange = vi.fn()
    const values = { ...baseValues(), physicalTool: 'walg' as const }
    render(<PhysicalMode values={values} onChange={onChange} />)

    fireEvent.click(screen.getByTestId('envs-suggest-WALG_S3_PREFIX'))
    expect(onChange).toHaveBeenLastCalledWith('physicalEnvs', [
      { key: 'WALG_S3_PREFIX', value: '' },
    ])
  })

  it('envs editor disables a suggestion already used in the list', () => {
    const values = {
      ...baseValues(),
      physicalTool: 'walg' as const,
      physicalEnvs: [{ key: 'WALG_S3_PREFIX', value: 's3://bucket' }],
    }
    render(<PhysicalMode values={values} onChange={vi.fn()} />)

    const button = screen.getByTestId('envs-suggest-WALG_S3_PREFIX') as HTMLButtonElement
    expect(button.disabled).toBe(true)
  })

  it('envs editor surfaces per-row keyErrors as helper text on both duplicate rows', () => {
    const values = {
      ...baseValues(),
      physicalTool: 'walg' as const,
      physicalEnvs: [
        { key: 'AWS_REGION', value: 'us-east-1' },
        { key: 'AWS_REGION', value: 'us-west-2' },
      ],
    }
    render(
      <PhysicalMode
        values={values}
        onChange={vi.fn()}
        envsKeyErrors={['Duplicate key', 'Duplicate key']}
      />,
    )

    const matches = screen.getAllByText('Duplicate key')
    expect(matches.length).toBeGreaterThanOrEqual(2)
  })

  it('envs editor shows no error when keyErrors is empty', () => {
    const values = {
      ...baseValues(),
      physicalTool: 'walg' as const,
      physicalEnvs: [
        { key: 'AWS_REGION', value: 'us-east-1' },
        { key: 'WALG_S3_PREFIX', value: 's3://bucket' },
      ],
    }
    render(
      <PhysicalMode
        values={values}
        onChange={vi.fn()}
        envsKeyErrors={[undefined, undefined]}
      />,
    )

    expect(screen.queryByText('Duplicate key')).not.toBeInTheDocument()
  })

  it('Sync section binds docker image and sync.enabled', () => {
    const onChange = vi.fn()
    const values = {
      ...baseValues(),
      physicalTool: 'walg' as const,
      physicalDockerImage: 'postgresai/extended-postgres:18-0.6.2',
    }
    render(<PhysicalMode values={values} onChange={onChange} />)

    const image = screen.getByTestId('physical-docker-image') as HTMLInputElement
    expect(image.value).toBe('postgresai/extended-postgres:18-0.6.2')

    fireEvent.change(image, { target: { value: 'postgresai/extended-postgres:17' } })
    expect(onChange).toHaveBeenCalledWith(
      'physicalDockerImage',
      'postgresai/extended-postgres:17',
    )

    const syncToggle = screen.getByTestId('physical-sync-enabled') as HTMLInputElement
    fireEvent.click(syncToggle)
    expect(onChange).toHaveBeenCalledWith('physicalSyncEnabled', true)
  })
})

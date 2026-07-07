import { describe, it, expect, vi } from 'vitest'
import { render, act } from '@testing-library/react'

import { useForm } from '@postgres.ai/shared/pages/Instance/Configuration/useForm'

type FormState = ReturnType<typeof useForm>[0]

const setupHook = () => {
  const submitSpy = vi.fn()
  const ref: { current: FormState | null } = { current: null }

  const Harness = () => {
    const [state] = useForm(submitSpy)
    ref.current = state
    return null
  }

  const utils = render(<Harness />)
  const api = () => ref.current as FormState
  return { api, ...utils, submitSpy }
}

describe('useForm — connection-string integration', () => {
  it('builds a postgres:// URI from current host/port/user/dbname', () => {
    const { api, rerender } = setupHook()

    act(() => {
      api().formik.setValues({
        ...api().formik.values,
        host: 'db.example.com',
        port: '5432',
        username: 'app',
        dbname: 'shop',
      })
    })
    rerender(<></>)

    expect(api().connectionString).toBe('postgres://app@db.example.com:5432/shop')
  })

  it('typing a new connection string updates the underlying fields', () => {
    const { api } = setupHook()

    act(() => {
      api().onConnectionStringChange('postgres://guest@new.example.com:6543/store')
    })

    expect(api().formik.values).toMatchObject({
      host: 'new.example.com',
      port: '6543',
      username: 'guest',
      dbname: 'store',
    })
    expect(api().connectionStringError).toBeNull()
  })

  it('surfaces a connection-string error without clobbering fields', () => {
    const { api } = setupHook()

    act(() => {
      api().formik.setValues({
        ...api().formik.values,
        host: 'db.example.com',
        port: '5432',
        username: 'app',
        dbname: 'shop',
      })
    })

    act(() => {
      api().onConnectionStringChange('postgres://app:secret@db.example.com/shop')
    })

    expect(api().connectionStringError).toMatch(/Password/i)
    expect(api().formik.values.host).toBe('db.example.com')
  })

  it('preserves empty port across load → URL → save', () => {
    const { api } = setupHook()

    act(() => {
      api().markPortInitialState(true)
      api().formik.setValues({
        ...api().formik.values,
        host: 'rds.example.com',
        port: '5432',
        username: 'app',
        dbname: 'shop',
      })
    })

    expect(api().connectionString).toBe('postgres://app@rds.example.com/shop')
    expect(api().omitPortOnSubmit).toBe(true)
  })

  it('marks port dirty when user types a URL with explicit port', () => {
    const { api } = setupHook()

    act(() => {
      api().markPortInitialState(true)
    })

    act(() => {
      api().onConnectionStringChange('postgres://app@host:6543/shop')
    })

    expect(api().omitPortOnSubmit).toBe(false)
    expect(api().connectionString).toBe('postgres://app@host:6543/shop')
  })

  it('keeps port omission when user types a URL without an explicit port', () => {
    const { api } = setupHook()

    act(() => {
      api().markPortInitialState(true)
    })

    act(() => {
      api().onConnectionStringChange('postgres://app@host/shop')
    })

    expect(api().omitPortOnSubmit).toBe(true)
    expect(api().connectionString).toBe('postgres://app@host/shop')
  })

  it('clears all four fields when the URL is emptied', () => {
    const { api } = setupHook()

    act(() => {
      api().onConnectionStringChange('postgres://app@host:5432/shop')
    })

    act(() => {
      api().onConnectionStringChange('   ')
    })

    expect(api().formik.values).toMatchObject({ host: '', port: '', username: '', dbname: '' })
  })
})

describe('useForm — physicalEnvs unique-keys validation', () => {
  const setPhysicalEnvs = async (api: ReturnType<typeof setupHook>['api'], envs: Array<{ key: string; value: string }>) => {
    await act(async () => {
      await api().formik.setFieldValue('retrievalMode', 'physical')
      await api().formik.setFieldValue('dockerImage', 'pg-image')
      await api().formik.setFieldValue('physicalEnvs', envs, true)
    })
    await act(async () => {
      await api().formik.validateForm()
    })
  }

  it('flags both rows when two keys collide', async () => {
    const { api } = setupHook()
    await setPhysicalEnvs(api, [
      { key: 'AWS_REGION', value: 'us-east-1' },
      { key: 'AWS_REGION', value: 'us-west-2' },
    ])

    const rowErrors = api().formik.errors.physicalEnvs as Array<{ key?: string } | undefined>
    expect(rowErrors?.[0]?.key).toBe('Duplicate key')
    expect(rowErrors?.[1]?.key).toBe('Duplicate key')
    expect(api().formik.isValid).toBe(false)
  })

  it('clears both errors once the duplicate is renamed', async () => {
    const { api } = setupHook()
    await setPhysicalEnvs(api, [
      { key: 'AWS_REGION', value: 'us-east-1' },
      { key: 'AWS_REGION', value: 'us-west-2' },
    ])
    expect((api().formik.errors.physicalEnvs as Array<{ key?: string } | undefined>)?.[0]?.key).toBe('Duplicate key')

    await setPhysicalEnvs(api, [
      { key: 'AWS_REGION', value: 'us-east-1' },
      { key: 'AWS_DEFAULT_REGION', value: 'us-west-2' },
    ])

    expect(api().formik.errors.physicalEnvs).toBeUndefined()
    expect(api().formik.isValid).toBe(true)
  })

  it('clears the remaining error when one of the duplicated rows is removed', async () => {
    const { api } = setupHook()
    await setPhysicalEnvs(api, [
      { key: 'AWS_REGION', value: 'us-east-1' },
      { key: 'AWS_REGION', value: 'us-west-2' },
    ])

    await setPhysicalEnvs(api, [{ key: 'AWS_REGION', value: 'us-east-1' }])

    expect(api().formik.errors.physicalEnvs).toBeUndefined()
    expect(api().formik.isValid).toBe(true)
  })

  it('does not flag rows with whitespace-only or empty keys as duplicates', async () => {
    const { api } = setupHook()
    await setPhysicalEnvs(api, [
      { key: '   ', value: 'a' },
      { key: '', value: 'b' },
      { key: '   ', value: 'c' },
    ])

    expect(api().formik.errors.physicalEnvs).toBeUndefined()
    expect(api().formik.isValid).toBe(true)
  })
})

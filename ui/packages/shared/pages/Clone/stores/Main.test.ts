import { describe, it, expect } from 'vitest'

import { Api, MainStore } from '@postgres.ai/shared/pages/Clone/stores/Main'
import { formatInstanceDto } from '@postgres.ai/shared/types/api/entities/instance'
import { InstanceStateDto } from '@postgres.ai/shared/types/api/entities/instanceState'

const notCalled = () => Promise.reject(new Error('not called'))

const buildApi = (overrides: Partial<Api> = {}): Api => ({
  getInstance: notCalled,
  getClone: notCalled,
  resetClone: notCalled,
  destroyClone: notCalled,
  updateClone: notCalled,
  ...overrides,
})

const buildInstance = (cloneUpgrade?: InstanceStateDto['cloneUpgrade']) =>
  formatInstanceDto({
    id: 1,
    plan: '',
    selfassigned_instance_id: '',
    state: {
      cloning: { clones: [], expectedCloningTime: 0 },
      status: { code: 'OK', message: '' },
      cloneUpgrade,
    },
  })

const buildStore = (
  cloneUpgrade?: InstanceStateDto['cloneUpgrade'],
  api: Api = buildApi({ upgradeClone: notCalled }),
) => {
  const store = new MainStore(api)
  store.instance = buildInstance(cloneUpgrade)
  return store
}

describe('MainStore clone upgrade gate', () => {
  it('hides the action when the api has no upgrade endpoint', () => {
    const store = buildStore({ available: true, targetVersion: 17 }, buildApi())

    expect(store.isUpgradeSupported).toBe(false)
    expect(store.upgradeTargetVersion).toBe(17)
  })

  it('hides the action until the instance is loaded', () => {
    const store = new MainStore(buildApi({ upgradeClone: notCalled }))

    expect(store.isUpgradeSupported).toBe(false)
    expect(store.upgradeUnavailableReason).toBeUndefined()
  })

  it('hides the action when the engine does not report it', () => {
    const store = buildStore(undefined)

    expect(store.isUpgradeSupported).toBe(false)
    expect(store.upgradeTargetVersion).toBeUndefined()
    expect(store.upgradeUnavailableReason).toBeUndefined()
  })

  it('keeps the action visible with the engine reason when unavailable', () => {
    const store = buildStore({
      available: false,
      reason: 'provision.pgUpgradeImage is not set',
    })

    expect(store.isUpgradeSupported).toBe(true)
    expect(store.upgradeTargetVersion).toBeUndefined()
    expect(store.upgradeUnavailableReason).toBe(
      'provision.pgUpgradeImage is not set',
    )
  })

  it('falls back to a generic reason when the engine gives none', () => {
    const store = buildStore({ available: false })

    expect(store.isUpgradeSupported).toBe(true)
    expect(store.upgradeUnavailableReason).toBe(
      'Clone upgrade is not available on this instance',
    )
  })

  it('enables the action with the target version when available', () => {
    const store = buildStore({ available: true, targetVersion: 17 })

    expect(store.isUpgradeSupported).toBe(true)
    expect(store.upgradeTargetVersion).toBe(17)
    expect(store.upgradeUnavailableReason).toBeUndefined()
  })
})

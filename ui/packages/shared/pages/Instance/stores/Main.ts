/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */
import { makeAutoObservable } from 'mobx'

import { GetSnapshots } from '@postgres.ai/shared/types/api/endpoints/getSnapshots'
import { GetInstance } from '@postgres.ai/shared/types/api/endpoints/getInstance'
import { RefreshInstance } from '@postgres.ai/shared/types/api/endpoints/refreshInstance'
import { DestroyClone } from '@postgres.ai/shared/types/api/endpoints/destroyClone'
import { ResetClone } from '@postgres.ai/shared/types/api/endpoints/resetClone'
import { Instance } from '@postgres.ai/shared/types/api/entities/instance'
import { SnapshotsStore } from '@postgres.ai/shared/stores/Snapshots'
import { getTextFromUnknownApiError } from '@postgres.ai/shared/utils/api'

const POLLING_TIME = 2000

const UNSTABLE_CLONE_STATUS_CODES = ['CREATING', 'RESETTING', 'DELETING']

export type Api = {
  getInstance: GetInstance
  getSnapshots: GetSnapshots
  refreshInstance?: RefreshInstance
  destroyClone: DestroyClone
  resetClone: ResetClone
}

type Error = {
  title?: string
  message: string
}

export class MainStore {
  instance: Instance | null = null
  instanceError: Error | null = null

  unstableClones = new Set<string>()
  private updateInstanceTimeoutId: number | null = null

  readonly snapshots: SnapshotsStore

  isReloadingClones = false

  private readonly api: Api

  constructor(api: Api) {
    makeAutoObservable(this)

    this.api = api
    this.snapshots = new SnapshotsStore(api)
  }

  get isDisabledInstance() {
    if (!this.instance) return true
    return this.instance.state.status.code === 'NO_RESPONSE'
  }

  load = (instanceId: string) => {
    this.instance = null

    this.loadInstance(instanceId)
    this.snapshots.load(instanceId)
  }

  reloadSnapshots = async () => {
    if (!this.instance) return
    await this.snapshots.reload(this.instance.id)
  }

  private loadInstance = async (
    instanceId: string,
    updateUnstableClones = true,
  ) => {
    this.instanceError = null

    if (this.api.refreshInstance)
      await this.api.refreshInstance({ instanceId: instanceId })

    const { response, error } = await this.api.getInstance({
      instanceId: instanceId,
    })

    if (response === null) {
      this.instanceError = {
        title: 'Error 404',
        message: 'Specified instance not found or you have no access.',
      }
    }

    if (response) {
      this.instance = response

      const unstableClones = new Set<string>()

      this.instance.state.cloning.clones.forEach((clone) => {
        if (UNSTABLE_CLONE_STATUS_CODES.includes(clone.status.code)) {
          unstableClones.add(clone.id)
        }
      })

      this.unstableClones = unstableClones

      if (this.unstableClones.size && updateUnstableClones)
        this.liveUpdateInstance()
    }

    if (error)
      this.instanceError = { message: await getTextFromUnknownApiError(error) }

    return !!response
  }

  resetClone = async (cloneId: string, snapshotId: string) => {
    if (!this.instance) return

    this.unstableClones.add(cloneId)

    const { response, error } = await this.api.resetClone({
      cloneId,
      snapshotId,
      instanceId: this.instance.id,
    })

    if (response) this.liveUpdateInstance()
    if (error) await getTextFromUnknownApiError(error)

    return !!response
  }

  destroyClone = async (cloneId: string) => {
    if (!this.instance) return

    this.unstableClones.add(cloneId)

    const { response, error } = await this.api.destroyClone({
      cloneId,
      instanceId: this.instance.id,
    })

    if (response) this.liveUpdateInstance()
    if (error) await getTextFromUnknownApiError(error)

    return !!response
  }

  private liveUpdateInstance = async () => {
    if (this.updateInstanceTimeoutId)
      window.clearTimeout(this.updateInstanceTimeoutId)
    if (!this.unstableClones.size) return
    if (!this.instance) return

    await this.loadInstance(this.instance.id, true)

    if (!this.unstableClones.size) return

    this.updateInstanceTimeoutId = window.setTimeout(
      this.liveUpdateInstance,
      POLLING_TIME,
    )
  }

  reloadClones = async () => {
    if (!this.instance) return
    this.isReloadingClones = true
    await this.loadInstance(this.instance.id)
    this.isReloadingClones = false
  }
}

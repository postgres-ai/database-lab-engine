/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeAutoObservable } from 'mobx'

import { GetInstance } from '@postgres.ai/shared/types/api/endpoints/getInstance'
import { GetClone } from '@postgres.ai/shared/types/api/endpoints/getClone'
import { ResetClone } from '@postgres.ai/shared/types/api/endpoints/resetClone'
import { DestroyClone } from '@postgres.ai/shared/types/api/endpoints/destroyClone'
import { UpdateClone } from '@postgres.ai/shared/types/api/endpoints/updateClone'
import {
  SnapshotsStore,
  SnapshotsApi,
} from '@postgres.ai/shared/stores/Snapshots'
import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
import { Instance } from '@postgres.ai/shared/types/api/entities/instance'
import { checkIsCloneStable } from '@postgres.ai/shared/utils/clone'
import { getTextFromUnknownApiError } from '@postgres.ai/shared/utils/api'
import { InitWS } from '@postgres.ai/shared/types/api/endpoints/initWS'

const UNSTABLE_CLONE_UPDATE_TIMEOUT = 1000

export type Api = SnapshotsApi & {
  getInstance: GetInstance
  getClone: GetClone
  resetClone: ResetClone
  destroyClone: DestroyClone
  updateClone: UpdateClone
  initWS?: InitWS
}

type Error = {
  title?: string
  message: string
}

export class MainStore {
  instance: Instance | null = null
  instanceError: Error | null = null

  clone: Clone | null = null
  cloneError: Error | null = null

  readonly snapshots: SnapshotsStore

  isResettingClone = false
  resetCloneError: string | null = null

  isDestroyingClone = false
  destroyCloneError: string | null = null

  isUpdatingClone = false
  updateCloneError: string | null = null

  isReloading = false

  private cloneUpdateTimeout?: number

  private readonly api: Api

  constructor(api: Api) {
    this.snapshots = new SnapshotsStore(api)
    this.api = api

    makeAutoObservable(this)
  }

  get isCloneStable() {
    if (!this.clone) return false
    return checkIsCloneStable(this.clone)
  }

  load = async (instanceId: string, cloneId: string) => {
    const [isInstanceOk, isCloneOk, isSnapshotsLoaded] = await Promise.all([
      this.loadInstance(instanceId),
      this.loadClone(instanceId, cloneId),
      this.snapshots.load(instanceId),
    ])

    return isInstanceOk && isCloneOk && isSnapshotsLoaded
  }

  reload = async () => {
    if (!this.instance || !this.clone) return false
    this.isReloading = true
    const isSuccess = await this.load(this.instance.id, this.clone.id)
    this.isReloading = false
    return isSuccess
  }

  private loadInstance = async (instanceId: string) => {
    const { response, error } = await this.api.getInstance({
      instanceId,
    })

    if (response) {
      this.instance = response
    } else {
      this.instanceError = {
        title: 'Error',
        message: `Instance "${instanceId}" not found`,
      }
    }

    if (error) {
      this.instanceError = {
        message: await getTextFromUnknownApiError(error),
      }
    }

    return Boolean(response)
  }

  private loadClone = async (instanceId: string, cloneId: string) => {
    window.clearTimeout(this.cloneUpdateTimeout)

    const { response, error } = await this.api.getClone({ instanceId, cloneId })

    if (response) {
      this.clone = response

      if (!this.isCloneStable)
        this.cloneUpdateTimeout = window.setTimeout(
          () => this.loadClone(instanceId, cloneId),
          UNSTABLE_CLONE_UPDATE_TIMEOUT,
        )
    }

    if (error) {
      if (error.status === 404) {
        this.cloneError = {
          title: 'Error',
          message: `Clone "${cloneId}" not found`,
        }
      } else {
        this.cloneError = {
          message: await getTextFromUnknownApiError(error),
        }
      }
    }

    return Boolean(response)
  }

  resetClone = async (snapshotId: string) => {
    if (!this.instance || !this.clone) return false

    this.isResettingClone = true

    const { response, error } = await this.api.resetClone({
      instanceId: this.instance.id,
      cloneId: this.clone.id,
      snapshotId,
    })

    if (response) await this.loadClone(this.instance.id, this.clone.id)

    if (error) this.resetCloneError = await getTextFromUnknownApiError(error)

    this.isResettingClone = false

    return Boolean(response)
  }

  destroyClone = async () => {
    if (!this.instance || !this.clone) return false

    this.isDestroyingClone = true

    const { response, error } = await this.api.destroyClone({
      instanceId: this.instance.id,
      cloneId: this.clone.id,
    })

    if (error) this.destroyCloneError = await getTextFromUnknownApiError(error)

    this.isDestroyingClone = false

    return Boolean(response)
  }

  updateCloneProtection = async (durationMinutes: number | null) => {
    if (!this.instance || !this.clone) return

    this.isUpdatingClone = true

    const prevIsProtected = this.clone.protected
    const isProtected = durationMinutes !== null

    this.clone.protected = isProtected

    const { response, error } = await this.api.updateClone({
      instanceId: this.instance.id,
      cloneId: this.clone.id,
      clone: {
        isProtected,
        protectionDurationMinutes: durationMinutes ?? undefined,
      },
    })

    if (response) {
      await this.loadClone(this.instance.id, this.clone.id)
    } else {
      this.clone.protected = prevIsProtected
    }

    if (error) this.updateCloneError = await getTextFromUnknownApiError(error)

    this.isUpdatingClone = false
  }
}

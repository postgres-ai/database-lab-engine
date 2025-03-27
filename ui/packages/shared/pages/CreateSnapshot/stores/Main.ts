/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeAutoObservable } from 'mobx'

import { CreateSnapshot } from '@postgres.ai/shared/types/api/endpoints/createSnapshot'
import {
  MainStore as InstanceStore,
  Api as InstanceStoreApi,
} from '@postgres.ai/shared/pages/Instance/stores/Main'

type Error = {
  title?: string
  message: string
}

export type MainStoreApi = InstanceStoreApi & {
  createSnapshot: CreateSnapshot
}

export class MainStore {
  snapshotError: Error | null = null

  isCreatingSnapshot = false

  readonly instance: InstanceStore

  private readonly api: MainStoreApi

  constructor(api: MainStoreApi) {
    this.api = api
    this.instance = new InstanceStore(api)

    makeAutoObservable(this)
  }

  load = async (instanceId: string) => {
    this.instance.load(instanceId)
  }

  createSnapshot = async (
    cloneID: string,
    message: string,
    instanceId: string,
  ) => {
    if (!this.api.createSnapshot) return

    this.snapshotError = null
    this.isCreatingSnapshot = true

    const { response, error } = await this.api.createSnapshot(
      cloneID,
      message,
      instanceId,
    )

    this.isCreatingSnapshot = false

    if (error)
      this.snapshotError = await error.json().then((err) => err.message)

    return response
  }
}

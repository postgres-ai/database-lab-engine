/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeAutoObservable } from 'mobx'

import { GetBranches } from '@postgres.ai/shared/types/api/endpoints/getBranches'
import {
  CreateBranch,
  CreateBranchFormValues,
} from '@postgres.ai/shared/types/api/endpoints/createBranch'
import { Branch } from '@postgres.ai/shared/types/api/endpoints/getBranches'
import { GetSnapshots } from '@postgres.ai/shared/types/api/endpoints/getSnapshots'
import { InitWS } from '@postgres.ai/shared/types/api/endpoints/initWS'

type Error = {
  title?: string
  message: string
}

export type MainStoreApi = {
  getBranches: GetBranches
  createBranch: CreateBranch
  getSnapshots: GetSnapshots
  initWS?: InitWS
}

export class MainStore {
  snapshotsError: Error | null = null
  getBranchesError: Error | null = null
  createBranchError: string | null = null

  isBranchesLoading = false
  isCreatingBranch = false

  branchesList: Branch[] = []
  private readonly api: MainStoreApi

  constructor(api: MainStoreApi) {
    this.api = api
    makeAutoObservable(this)
  }

  load = async (instanceId: string) => {
    await this.getBranches(instanceId).then((response) => {
      if (response) {
        this.branchesList = response
      }
    })
  }

  createBranch = async (values: CreateBranchFormValues) => {
    if (!this.api.createBranch) return

    this.isCreatingBranch = true
    this.createBranchError = null

    const { response, error } = await this.api.createBranch(values)

    this.isCreatingBranch = false

    if (error)
      this.createBranchError = await error.json().then((err) => err.details)

    return response
  }

  getBranches = async (instanceId: string) => {
    if (!this.api.getBranches) return
    this.isBranchesLoading = true

    const { response, error } = await this.api.getBranches(instanceId)

    this.isBranchesLoading = false

    if (error) this.getBranchesError = await error.json().then((err) => err)

    return response
  }

  getSnapshots = async (instanceId: string, branchName?: string, dataset?: string) => {
    if (!this.api.getSnapshots) return

    const { response, error } = await this.api.getSnapshots({
      instanceId,
      branchName,
      dataset,
    })

    if (error) {
      this.snapshotsError = await error.json().then((err) => err)
    }

    return response
  }
}

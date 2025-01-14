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
import { GetBranchSnapshots } from 'types/api/endpoints/getBranchSnapshots'

type Error = {
  title?: string
  message: string
}

export type MainStoreApi = {
  getBranches: GetBranches
  createBranch: CreateBranch
  getBranchSnapshots: GetBranchSnapshots
}

export class MainStore {
  branchSnapshotsError: Error | null = null
  getBranchesError: Error | null = null
  createBranchError: Error | null = null

  isBranchesLoading = false
  isCreatingBranch = false

  branchesList: Branch[] = []
  private readonly api: MainStoreApi

  constructor(api: MainStoreApi) {
    this.api = api
    makeAutoObservable(this)
  }

  load = async () => {
    await this.getBranches().then((response) => {
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

    if (error) this.createBranchError = await error.json().then((err) => err)

    return response
  }

  getBranches = async () => {
    if (!this.api.getBranches) return
    this.isBranchesLoading = true

    const { response, error } = await this.api.getBranches()

    this.isBranchesLoading = false

    if (error) this.getBranchesError = await error.json().then((err) => err)

    return response
  }

  getBranchSnapshots = async (branchName: string) => {
    if (!this.api.getBranchSnapshots) return

    const { response, error } = await this.api.getBranchSnapshots(branchName)

    if (error) {
      this.branchSnapshotsError = await error.json().then((err) => err)
    }

    return response
  }
}

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
import {
  GetSnapshotList,
  GetSnapshotListResponseType,
} from '@postgres.ai/shared/types/api/endpoints/getSnapshotList'
import { GetBranchesResponseType } from '@postgres.ai/shared/types/api/endpoints/getBranches'

type Error = {
  title?: string
  message: string
}

export type MainStoreApi = {
  getBranches: GetBranches
  createBranch: CreateBranch
  getSnapshotList: GetSnapshotList
}

export class MainStore {
  snapshotListError: Error | null = null
  getBranchesError: Error | null = null
  createBranchError: Error | null = null

  isBranchesLoading = false
  isCreatingBranch = false

  branchesList: GetBranchesResponseType[] = []
  snapshotsList: GetSnapshotListResponseType[] = []
  private readonly api: MainStoreApi

  constructor(api: MainStoreApi) {
    this.api = api
    makeAutoObservable(this)
  }

  load = async (baseBranch: string) => {
    await this.getBranches()
      .then((response) => {
        if (response) {
          this.branchesList = response
        }
      })
      .then(() => {
        this.getSnapshotList(baseBranch).then((res) => {
          if (res) {
            const filteredSnapshots = res.filter((snapshot) => snapshot.id)
            this.snapshotsList = filteredSnapshots
          }
        })
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

    if (error) this.getBranchesError = await error.json().then((err) => err)

    return response
  }

  getSnapshotList = async (branchName: string) => {
    if (!this.api.getSnapshotList) return

    const { response, error } = await this.api.getSnapshotList(branchName)

    this.isBranchesLoading = false

    if (error) {
      this.snapshotListError = await error.json().then((err) => err)
    }

    return response
  }
}

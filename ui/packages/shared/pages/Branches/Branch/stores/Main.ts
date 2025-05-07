/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeAutoObservable } from 'mobx'

import { GetBranches } from '@postgres.ai/shared/types/api/endpoints/getBranches'
import { Branch } from '@postgres.ai/shared/types/api/endpoints/getBranches'
import { DeleteBranch } from '@postgres.ai/shared/types/api/endpoints/deleteBranch'
import {
  SnapshotList,
  GetSnapshotList,
} from '@postgres.ai/shared/types/api/endpoints/getSnapshotList'
import { InitWS } from '@postgres.ai/shared/types/api/endpoints/initWS'

type Error = {
  title?: string
  message: string
}

export type Api = {
  getBranches: GetBranches
  deleteBranch: DeleteBranch
  getSnapshotList: GetSnapshotList
  initWS?: InitWS
}

export class MainStore {
  getBranchError: Error | null = null
  snapshotListError: Error | null = null
  getBranchesError: Error | null = null

  isReloading = false
  isBranchesLoading = false

  branches: Branch[] | null = null
  branch: Branch | null = null
  snapshotList: SnapshotList[] | null = null

  private readonly api: Api

  constructor(api: Api) {
    this.api = api
    makeAutoObservable(this)
  }

  load = async (branchId: string, instanceId: string) => {
    if (!branchId) return

    this.isBranchesLoading = true

    await this.getBranches(branchId, instanceId)
  }

  reload = async (branchId: string, instanceId: string) => {
    if (!branchId) return

    this.isReloading = true
    await this.getBranches(branchId, instanceId)
    this.isReloading = false
  }

  getBranches = async (branchId: string, instanceId: string) => {
    if (!this.api.getBranches) return
    const { response, error } = await this.api.getBranches(instanceId)

    if (error) {
      this.isBranchesLoading = false
      this.getBranchesError = await error.json().then((err) => err)
    }

    if (response) {
      this.branches = response
      this.getBranch(branchId, instanceId)
    }

    return response
  }

  getBranch = async (branchId: string, instanceId: string) => {
    const currentBranch = this.branches?.filter((s) => {
      return s.name === branchId
    })

    if (currentBranch && currentBranch?.length > 0) {
      this.branch = currentBranch[0]
      this.getSnapshotList(currentBranch[0].name, instanceId)
    } else {
      this.getBranchError = {
        title: 'Error',
        message: `Branch "${branchId}" not found`,
      }
      this.isBranchesLoading = false
    }

    return !!currentBranch
  }

  deleteBranch = async (branchName: string, instanceId: string) => {
    if (!branchName) return

    const { response, error } = await this.api.deleteBranch(
      branchName,
      instanceId,
    )

    if (response) {
      this.branches =
        this.branches?.filter((branch) => branch.name !== branchName) || []
    }

    return { response, error }
  }

  getSnapshotList = async (branchName: string, instanceId: string) => {
    if (!this.api.getSnapshotList) return

    const { response, error } = await this.api.getSnapshotList(
      branchName,
      instanceId,
    )

    this.isBranchesLoading = false

    if (error) {
      this.snapshotListError = await error.json().then((err) => err)
    }

    if (response) {
      this.snapshotList = response
    }

    return response
  }
}

/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeAutoObservable } from 'mobx'

import {
  SnapshotsStore,
  SnapshotsApi,
} from '@postgres.ai/shared/stores/Snapshots'
import { DestroySnapshot } from '@postgres.ai/shared/types/api/endpoints/destroySnapshot'
import { SnapshotDto } from '@postgres.ai/shared/types/api/entities/snapshot'
import { GetBranchSnapshot } from '@postgres.ai/shared/types/api/endpoints/getBranchSnapshot'
import { BranchSnapshotDto } from '@postgres.ai/shared/types/api/entities/branchSnapshot'
import { generateSnapshotPageId } from '@postgres.ai/shared/pages/Instance/Snapshots/utils'
import { InitWS } from '@postgres.ai/shared/types/api/endpoints/initWS'

type Error = {
  title?: string
  message: string
}

export type Api = SnapshotsApi & {
  destroySnapshot: DestroySnapshot
  getBranchSnapshot?: GetBranchSnapshot
  initWS?: InitWS
}

export class MainStore {
  snapshot: SnapshotDto | null = null
  branchSnapshot: BranchSnapshotDto | null = null

  snapshotError: Error | null = null
  branchSnapshotError: Error | null = null

  isSnapshotsLoading = false

  private readonly api: Api
  readonly snapshots: SnapshotsStore

  constructor(api: Api) {
    this.api = api
    this.snapshots = new SnapshotsStore(api)
    makeAutoObservable(this)
  }

  load = async (snapshotId: string, instanceId: string) => {
    if (!snapshotId) return

    this.isSnapshotsLoading = true

    await this.snapshots.load(instanceId).then((loaded) => {
      loaded && this.getSnapshot(snapshotId, instanceId)
    })
  }
  getSnapshot = async (snapshotId: string, instanceId: string) => {
    if (!snapshotId) return

    const allSnapshots = this.snapshots.data
    const snapshot = allSnapshots?.filter((s: SnapshotDto) => {
      return snapshotId === generateSnapshotPageId(s.id)
    })

    if (snapshot && snapshot?.length > 0) {
      this.snapshot = snapshot[0]
      this.getBranchSnapshot(snapshot[0].id, instanceId)
    } else {
      this.isSnapshotsLoading = false
      this.snapshotError = {
        title: 'Error',
        message: `Snapshot "${snapshotId}" not found`,
      }
    }

    return !!snapshot
  }

  getBranchSnapshot = async (snapshotId: string, instanceId: string) => {
    if (!snapshotId || !this.api.getBranchSnapshot) return

    const { response, error } = await this.api.getBranchSnapshot(
      snapshotId,
      instanceId,
    )

    this.isSnapshotsLoading = false

    if (error) {
      this.branchSnapshotError = await error.json().then((err) => err)
    }

    if (response) {
      this.branchSnapshot = response
    }

    return response
  }

  destroySnapshot = async (
    snapshotId: string,
    forceDelete: boolean,
    instanceId: string,
  ) => {
    if (!this.api.destroySnapshot || !snapshotId) return

    const { response, error } = await this.api.destroySnapshot(
      snapshotId,
      forceDelete,
      instanceId,
    )

    return {
      response,
      error: error ? await error.json().then((err) => err) : null,
    }
  }
}

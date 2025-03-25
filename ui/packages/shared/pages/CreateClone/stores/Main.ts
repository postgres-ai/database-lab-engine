import { makeAutoObservable } from 'mobx'

import { Instance } from '@postgres.ai/shared/types/api/entities/instance'
import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
import { GetInstance } from '@postgres.ai/shared/types/api/endpoints/getInstance'
import { CreateClone } from '@postgres.ai/shared/types/api/endpoints/createClone'
import { GetClone } from '@postgres.ai/shared/types/api/endpoints/getClone'
import { GetBranches } from '@postgres.ai/shared/types/api/endpoints/getBranches'
import { GetSnapshots } from '@postgres.ai/shared/types/api/endpoints/getSnapshots'
import {
  SnapshotsStore,
  SnapshotsApi,
} from '@postgres.ai/shared/stores/Snapshots'
import { getTextFromUnknownApiError } from '@postgres.ai/shared/utils/api'
import { checkIsCloneStable } from '@postgres.ai/shared/utils/clone'

import { FormValues } from '../useForm'

const UNSTABLE_CLONE_UPDATE_TIMEOUT = 1000

export type MainStoreApi = SnapshotsApi & {
  getInstance: GetInstance
  createClone: CreateClone
  getClone: GetClone
  getBranches?: GetBranches
  getSnapshots?: GetSnapshots
}

export class MainStore {
  instance: Instance | null = null
  instanceError: string | null = null
  getBranchesError: Error | null = null
  getSnapshotsError: Error | null = null

  clone: Clone | null = null
  cloneError: string | null = null

  private cloneUpdateTimeout?: number

  private readonly api: MainStoreApi

  readonly snapshots?: SnapshotsStore

  constructor(api: MainStoreApi) {
    makeAutoObservable(this)

    this.api = api
    if (!api.getSnapshots) {
      this.snapshots = new SnapshotsStore(api)
    }
  }

  get isCloneStable() {
    if (!this.clone) return
    return checkIsCloneStable(this.clone)
  }

  load = async (instanceId: string) => {
    const [instance, isLoadedSnapshots] = await Promise.all([
      this.api.getInstance({ instanceId }),
      this.snapshots?.load(instanceId) ?? true,
    ])

    if (instance.response) this.instance = instance.response
    if (instance.error)
      this.instanceError = await getTextFromUnknownApiError(instance.error)

    return Boolean(instance.response) && isLoadedSnapshots
  }

  createClone = async (data: FormValues) => {
    if (!this.instance) return false

    const { response, error } = await this.api.createClone({
      ...data,
      instanceId: this.instance.id,
    })

    if (response) {
      this.clone = response

      this.updateCloneUntilStable({
        instanceId: this.instance.id,
        cloneId: this.clone.id,
      })
    }

    if (error)
      this.cloneError = await error.json().then((err) => err?.message || err)

    return Boolean(response)
  }

  getBranches = async () => {
    if (!this.api.getBranches) return
    const { response, error } = await this.api.getBranches()

    if (error) this.getBranchesError = await error.json().then((err) => err)

    return response
  }

  getSnapshots = async (instanceId: string, branchName?: string) => {
    if (!this.api.getSnapshots) return
    const { response, error } = await this.api.getSnapshots({
      instanceId,
      branchName,
    })

    if (error) this.getSnapshotsError = await error.json().then((err) => err)

    return response
  }

  private updateCloneUntilStable = async (args: {
    instanceId: string
    cloneId: string
  }) => {
    window.clearTimeout(this.cloneUpdateTimeout)

    const { response, error } = await this.api.getClone(args)

    if (response) {
      this.clone = response

      if (!this.isCloneStable)
        this.cloneUpdateTimeout = window.setTimeout(
          () => this.updateCloneUntilStable(args),
          UNSTABLE_CLONE_UPDATE_TIMEOUT,
        )
    }

    if (error)
      this.cloneError = await error.json().then((err) => err?.message || err)
  }
}

/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */
import { makeAutoObservable } from 'mobx'

import { GetSnapshots } from '@postgres.ai/shared/types/api/endpoints/getSnapshots'
import { CreateSnapshot } from '@postgres.ai/shared/types/api/endpoints/createSnapshot'
import { GetInstance } from '@postgres.ai/shared/types/api/endpoints/getInstance'
import { Config } from '@postgres.ai/shared/types/api/entities/config'
import { GetConfig } from '@postgres.ai/shared/types/api/endpoints/getConfig'
import { UpdateConfig } from '@postgres.ai/shared/types/api/endpoints/updateConfig'
import { TestDbSource } from '@postgres.ai/shared/types/api/endpoints/testDbSource'
import { RefreshInstance } from '@postgres.ai/shared/types/api/endpoints/refreshInstance'
import { DestroyClone } from '@postgres.ai/shared/types/api/endpoints/destroyClone'
import { ResetClone } from '@postgres.ai/shared/types/api/endpoints/resetClone'
import { GetWSToken } from '@postgres.ai/shared/types/api/endpoints/getWSToken'
import { InitWS } from '@postgres.ai/shared/types/api/endpoints/initWS'
import { Instance } from '@postgres.ai/shared/types/api/entities/instance'
import { SnapshotsStore } from '@postgres.ai/shared/stores/Snapshots'
import { getTextFromUnknownApiError } from '@postgres.ai/shared/utils/api'
import { dbSource } from '@postgres.ai/shared/types/api/entities/dbSource'
import { GetFullConfig } from '@postgres.ai/shared/types/api/endpoints/getFullConfig'
import { GetInstanceRetrieval } from '@postgres.ai/shared/types/api/endpoints/getInstanceRetrieval'
import { InstanceRetrievalType } from '@postgres.ai/shared/types/api/entities/instanceRetrieval'
import { GetEngine } from '@postgres.ai/shared/types/api/endpoints/getEngine'
import { GetSnapshotList } from '@postgres.ai/shared/types/api/endpoints/getSnapshotList'
import { GetBranches } from '@postgres.ai/shared/types/api/endpoints/getBranches'
import { DeleteBranch } from '@postgres.ai/shared/types/api/endpoints/deleteBranch'
import { GetSeImages } from '@postgres.ai/shared/types/api/endpoints/getSeImages'
import { DestroySnapshot } from '@postgres.ai/shared/types/api/endpoints/destroySnapshot'
import { FullRefresh } from "../../../types/api/endpoints/fullRefresh";

const UNSTABLE_CLONE_STATUS_CODES = ['CREATING', 'RESETTING', 'DELETING']

export type Api = {
  getInstance: GetInstance
  getSnapshots?: GetSnapshots
  createSnapshot?: CreateSnapshot
  refreshInstance?: RefreshInstance
  destroyClone?: DestroyClone
  resetClone?: ResetClone
  getWSToken?: GetWSToken
  initWS?: InitWS
  getConfig?: GetConfig
  updateConfig?: UpdateConfig
  testDbSource?: TestDbSource
  getFullConfig?: GetFullConfig
  getSeImages?: GetSeImages
  getEngine?: GetEngine
  getInstanceRetrieval?: GetInstanceRetrieval
  getBranches?: GetBranches
  getSnapshotList?: GetSnapshotList
  deleteBranch?: DeleteBranch
  destroySnapshot?: DestroySnapshot
  fullRefresh?: FullRefresh
}

type Error = {
  title?: string
  message?: string
}

export class MainStore {
  instance: Instance | null = null
  instanceRetrieval: InstanceRetrievalType | null = null
  config: Config | null = null
  fullConfig?: string
  dleEdition?: string
  platformUrl?: string
  uiVersion?: string
  instanceError: Error | null = null
  configError: string | null = null
  getFullConfigError: string | null = null
  getBranchesError: Error | null = null
  snapshotListError: string | null = null
  seImagesError: string | undefined | null = null

  unstableClones = new Set<string>()

  readonly snapshots: SnapshotsStore

  isReloadingClones = false
  isConfigurationLoading = false
  isReloadingInstance = false
  isReloadingInstanceRetrieval = false
  isBranchesLoading = false
  isConfigLoading = false
  isLoadingInstance = false
  isLoadingInstanceRetrieval = false

  private readonly api: Api

  constructor(api: Api) {
    makeAutoObservable(this)

    this.api = api
    this.snapshots = new SnapshotsStore(api)
  }

  get isDisabledInstance() {
    if (!this.instance) return true
    return this.instance.state?.status.code === 'NO_RESPONSE'
  }

  load = (instanceId: string, isPlatform: boolean = false) => {
    this.instance = null
    this.instanceRetrieval = null
    this.isReloadingInstance = true
    this.isLoadingInstanceRetrieval = false

    if (!isPlatform) {
      this.getBranches(instanceId)
    }

    const runRetrieval = () => {
      this.loadInstanceRetrieval(instanceId).then(() => {
        if (this.instanceRetrieval) {
          this.getConfig(instanceId)
          this.getFullConfig(instanceId)
        }
      })
    }

    this.loadInstance(instanceId, false).then(() => {
      if (
        (this.instance?.createdAt && this.instance?.url) ||
        !this.instance?.createdAt
      ) {
        this.snapshots.load(instanceId)
      }

      if (isPlatform && this.instance?.url) {
        this.getBranches(instanceId)
        runRetrieval()
      }
    })

    if (!isPlatform) {
      runRetrieval()
    }
  }

  reload = (instanceId: string) => {
    this.instance = null
    this.instanceRetrieval = null
    this.isReloadingInstance = true
    this.isLoadingInstanceRetrieval = false
    
    this.loadInstance(instanceId, false).then(() => {
      if (this.api.refreshInstance)
        this.api.refreshInstance({ instanceId: instanceId })

      if (
        (this.instance?.createdAt && this.instance?.url) ||
        !this.instance?.createdAt
      ) {
        this.snapshots.load(instanceId)
      }
    })
    
    this.loadInstanceRetrieval(instanceId).then(() => {
      if (this.instanceRetrieval) {
        this.getConfig(instanceId)
        this.getFullConfig(instanceId)
      }
    })
  }

  reloadSnapshots = async (branchName?: string) => {
    if (!this.instance) return
    await this.snapshots.reload(this.instance.id, branchName)
  }

  reloadInstanceRetrieval = async () => {
    if (!this.instance) return
    this.isReloadingInstanceRetrieval = true
    this.loadInstanceRetrieval(this.instance.id)
    this.isReloadingInstanceRetrieval = false
  }

  private loadInstanceRetrieval = async (instanceId: string) => {
    if (!this.api.getInstanceRetrieval) {
      this.isLoadingInstanceRetrieval = false
      return
    }

    this.isLoadingInstanceRetrieval = true

    const { response, error } = await this.api.getInstanceRetrieval({
      instanceId: instanceId,
    })

    this.isLoadingInstanceRetrieval = false

    if (response) this.instanceRetrieval = response

    if (error)
      this.instanceError = { message: await getTextFromUnknownApiError(error) }

    return !!response
  }

  private loadInstance = async (
    instanceId: string,
    refresh: boolean = true,
  ) => {
    this.instanceError = null
    this.isLoadingInstance = true

    if (this.api.refreshInstance && refresh)
      await this.api.refreshInstance({ instanceId: instanceId })

    const { response, error } = await this.api.getInstance({
      instanceId: instanceId,
    })

    this.isLoadingInstance = false

    if (response === null) {
      this.instanceError = {
        title: 'Error 404',
        message: 'Specified instance not found or you have no access.',
      }
    }

    if (response) {
      this.instance = response

      const unstableClones = new Set<string>()

      this.instance.state?.cloning.clones?.forEach((clone) => {
        if (UNSTABLE_CLONE_STATUS_CODES.includes(clone.status.code)) {
          unstableClones.add(clone.id)
        }
      })

      this.unstableClones = unstableClones
    }

    if (error)
      this.instanceError = { message: await getTextFromUnknownApiError(error) }

    return !!response
  }

  getConfig = async (instanceId: string) => {
    if (!this.api.getConfig) return

    this.isConfigurationLoading = true
    this.isConfigLoading = true

    const { response, error } = await this.api.getConfig(instanceId)

    this.isConfigurationLoading = false
    this.isConfigLoading = false

    if (response) {
      this.config = response
      this.configError = null
    }

    if (error) {
      this.configError = await error.json().then((err) => err.message)
    }

    return response
  }

  updateConfig = async (values: Config, instanceId: string) => {
    if (!this.api.updateConfig) return

    const { response, error } = await this.api.updateConfig(
      { ...values },
      instanceId,
    )

    if (error) this.configError = await error.json().then((err) => err.message)

    return response
  }

  getFullConfig = async (instanceId: string) => {
    if (!this.api.getFullConfig) return

    const { response, error } = await this.api.getFullConfig(instanceId)

    if (response) {
      this.fullConfig = response

      const splitYML = this.fullConfig.split('---')
      this.platformUrl = splitYML[0]?.split('url: ')[1]?.split('\n')[0]
      this.uiVersion = splitYML[0]
        ?.split('dockerImage: "postgresai/ce-ui:')[2]
        ?.split('\n')[0]
        ?.replace(/['"]+/g, '')
    }

    if (error)
      this.getFullConfigError = await error
        .json()
        .then((err: Error) => err.message)

    return response
  }

  getSeImages = async (values: { packageGroup: string }) => {
    if (!this.api.getSeImages || !this.platformUrl) return

    const { response, error } = await this.api.getSeImages({
      packageGroup: values.packageGroup,
      platformUrl: this.platformUrl,
    })

    if (response) {
      this.seImagesError = null
    }

    if (error) {
      this.seImagesError = await error.json().then((err: Error) => err.message)
    }

    return response
  }

  getEngine = async (instanceId: string) => {
    if (!this.api.getEngine) return

    this.configError = null

    const { response, error } = await this.api.getEngine(instanceId)

    if (response) {
      this.dleEdition = response.edition
    }

    if (error) await getTextFromUnknownApiError(error)
    return response
  }

  testDbSource = async (values: dbSource) => {
    if (!this.api.testDbSource) return

    const { response, error } = await this.api.testDbSource(values)

    return {
      response,
      error,
    }
  }

  resetClone = async (cloneId: string, snapshotId: string) => {
    if (!this.instance || !this.api.resetClone) return

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
    if (!this.instance || !this.api.destroyClone) return

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
    if (!this.unstableClones.size) return
    if (!this.instance) return

    await this.loadInstance(this.instance.id)
    await this.loadInstanceRetrieval(this.instance.id)

    if (!this.unstableClones.size) return
  }

  reloadClones = async () => {
    if (!this.instance) return
    this.isReloadingClones = true
    await this.loadInstance(this.instance.id)
    await this.loadInstanceRetrieval(this.instance.id)
    this.isReloadingClones = false
  }

  getBranches = async (instanceId: string) => {
    if (!this.api.getBranches) return
    this.isBranchesLoading = true

    const { response, error } = await this.api.getBranches(instanceId)

    this.isBranchesLoading = false

    if (error) this.getBranchesError = await error.json().then((err) => err)

    return response
  }

  deleteBranch = async (branchName: string, instanceId: string) => {
    if (!branchName || !this.api.deleteBranch) return

    const { response, error } = await this.api.deleteBranch(
      branchName,
      instanceId,
    )

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
      this.snapshotListError = await error.json().then((err) => err.message)
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

  fullRefresh = async (instanceId: string): Promise<{ response: string | null, error: Error | null } | undefined> => {
    if (!this.api.fullRefresh) return

    const { response, error } = await this.api.fullRefresh({
      instanceId,
    })

    if (error) {
      const parsedError = await error.json().then((err) => ({
        message: err.message || 'An unknown error occurred',
      }));

      return { response: null, error: parsedError }
    } else if (this.instance?.state?.retrieving) {
      this.instance.state.retrieving.status = 'refreshing';
    }

    return { response: response ? String(response) : null, error: null }
  }
}

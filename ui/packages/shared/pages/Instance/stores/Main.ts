/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */
import { makeAutoObservable } from 'mobx'

import { GetSnapshots } from '@postgres.ai/shared/types/api/endpoints/getSnapshots'
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
import { GetSeImages } from '@postgres.ai/shared/types/api/endpoints/getSeImages'

const UNSTABLE_CLONE_STATUS_CODES = ['CREATING', 'RESETTING', 'DELETING']

export type Api = {
  getInstance: GetInstance
  getSnapshots: GetSnapshots
  refreshInstance?: RefreshInstance
  destroyClone: DestroyClone
  resetClone: ResetClone
  getWSToken: GetWSToken
  initWS?: InitWS
  getConfig?: GetConfig
  updateConfig?: UpdateConfig
  testDbSource?: TestDbSource
  getFullConfig?: GetFullConfig
  getSeImages?: GetSeImages
  getEngine?: GetEngine
  getInstanceRetrieval?: GetInstanceRetrieval
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
  platformUrl?: string
  instanceError: Error | null = null
  configError: string | null = null
  getFullConfigError: string | null = null
  seImagesError: string | undefined | null = null

  unstableClones = new Set<string>()

  readonly snapshots: SnapshotsStore

  isReloadingClones = false
  isConfigurationLoading = false
  isReloadingInstance = false
  isReloadingInstanceRetrieval = false
  isLoadingInstance = false

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
    this.isReloadingInstance = true
    this.loadInstance(instanceId, false)
    this.loadInstanceRetrieval(instanceId).then(() => {
      if (this.instanceRetrieval) {
        this.getConfig()
      }
    })
    this.snapshots.load(instanceId)
  }

  reload = (instanceId: string) => {
    this.instance = null
    this.isReloadingInstance = true
    this.loadInstance(instanceId)
    this.loadInstanceRetrieval(instanceId).then(() => {
      if (this.instanceRetrieval) {
        this.getConfig()
        this.getFullConfig()
      }
    })
    this.snapshots.load(instanceId)
  }

  reloadSnapshots = async () => {
    if (!this.instance) return
    await this.snapshots.reload(this.instance.id)
  }

  reloadInstanceRetrieval = async () => {
    if (!this.instance) return
    this.isReloadingInstanceRetrieval = true
    this.loadInstanceRetrieval(this.instance.id)
    this.isReloadingInstanceRetrieval = false
  }

  private loadInstanceRetrieval = async (instanceId: string) => {
    if (!this.api.getInstanceRetrieval) return

    const { response, error } = await this.api.getInstanceRetrieval({
      instanceId: instanceId,
    })

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

      this.instance.state.cloning.clones.forEach((clone) => {
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

  getConfig = async () => {
    if (!this.api.getConfig) return

    this.isConfigurationLoading = true

    const { response, error } = await this.api.getConfig()

    this.isConfigurationLoading = false

    if (response) {
      this.config = response
      this.configError = null
    }

    if (error) {
      this.configError = await error.json().then((err) => err.message)
    }

    return response
  }

  updateConfig = async (values: Config) => {
    if (!this.api.updateConfig) return

    const { response, error } = await this.api.updateConfig({ ...values })

    if (error) this.configError = await error.json().then((err) => err.message)

    return response
  }

  getFullConfig = async () => {
    if (!this.api.getFullConfig) return

    const { response, error } = await this.api.getFullConfig()
    if (response) {
      this.fullConfig = response

      const splitYML = this.fullConfig.split('---')
      this.platformUrl = splitYML[0]?.split('url: ')[1]?.split('\n')[0]
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

  getEngine = async () => {
    if (!this.api.getEngine) return

    this.configError = null

    const { response, error } = await this.api.getEngine()

    if (error) await getTextFromUnknownApiError(error)
    return response
  }

  testDbSource = async (values: dbSource) => {
    if (!this.api.testDbSource) return

    const { response, error } = await this.api.testDbSource(values)

    return {
      response,
      error
    }
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
}

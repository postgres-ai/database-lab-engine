import { makeAutoObservable } from 'mobx'

import { getMeta } from 'api/getMeta'
import { BUILD_TIMESTAMP } from 'config/env'

// 60 sec.
const META_INFO_POLLING_PERIOD = 60 * 1000

class AppStore {
  isOutdatedVersion = false

  constructor() {
    makeAutoObservable(this)

    this.checkMetaInfo()
  }

  private loadMetaInfo = async () => {
    const { response } = await getMeta()
    if (!response) return

    this.isOutdatedVersion = response.buildTimestamp > BUILD_TIMESTAMP
  }

  private checkMetaInfo = async () => {
    await this.loadMetaInfo()
    if (this.isOutdatedVersion) return
    setTimeout(this.checkMetaInfo, META_INFO_POLLING_PERIOD)
  }
}

export const appStore = new AppStore()

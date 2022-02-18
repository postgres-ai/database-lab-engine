import { makeAutoObservable } from 'mobx'

class BannersStore {
  isOpenDeprecatedApi = false

  constructor() {
    makeAutoObservable(this)
  }

  showDeprecatedApi = () => {
    this.isOpenDeprecatedApi = true
  }

  hideDeprecatedApi = () => {
    this.isOpenDeprecatedApi = false
  }
}

export const bannersStore = new BannersStore()

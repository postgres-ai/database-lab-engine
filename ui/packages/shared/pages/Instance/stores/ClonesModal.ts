/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeAutoObservable } from 'mobx'

type FilterOptions = {
  pool?: string | null
  snapshotId?: string
}

export class ClonesModalStore {
  isOpenModal = false
  pool: string | null = null
  snapshotId: string | null = null

  constructor() {
    makeAutoObservable(this)
  }

  openModal = (filterOptions: FilterOptions | undefined = {}) => {
    const { pool = null, snapshotId = null } = filterOptions

    this.pool = pool
    this.snapshotId = snapshotId
    this.isOpenModal = true
  }

  closeModal = () => {
    this.isOpenModal = false
  }
}

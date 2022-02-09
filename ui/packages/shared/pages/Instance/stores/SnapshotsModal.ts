/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeAutoObservable } from 'mobx'

type FilterOptions = {
  pool?: string | null
  date?: Date
}

export class SnapshotsModalStore {
  isOpenModal = false
  pool: string | null = null
  date: Date | null = null

  constructor() {
    makeAutoObservable(this)
  }

  openModal = (filterOptions: FilterOptions | undefined = {}) => {
    const { pool = null, date = null } = filterOptions

    this.pool = pool
    this.date = date
    this.isOpenModal = true
  }

  closeModal = () => {
    this.isOpenModal = false
  }
}

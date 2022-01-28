/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

 export type PoolDto = {
  cloneList: string[]
  fileSystem: {
    compressRatio: number
    dataSize: number
    free: number
    mode: string
    size: number
    used: number
    usedByClones: number
    usedBySnapshots: number
  }
  mode: string
  name: string
  status: 'active' | 'empty' | 'refreshing'
}

export const formatPoolDto = (dto: PoolDto) => dto

export type Pool = ReturnType<typeof formatPoolDto>

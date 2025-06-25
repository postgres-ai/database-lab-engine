/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Snapshot } from '@postgres.ai/shared/types/api/entities/snapshot'

export const getEdgeSnapshots = (snapshots: Snapshot[]) => {
  if (!snapshots.length) {
    return {
      firstSnapshot: null,
      lastSnapshot: null
    }
  }
  
  const sortedList = [...snapshots].sort((a, b) => {
    const aTime = a.dataStateAtDate?.getTime() ?? 0
    const bTime = b.dataStateAtDate?.getTime() ?? 0
    return bTime - aTime
  })

  const [first] = sortedList
  const [last] = sortedList.slice(-1)

  return {
    firstSnapshot: first ?? null, // newest
    lastSnapshot: last ?? null    // oldest
  }
}

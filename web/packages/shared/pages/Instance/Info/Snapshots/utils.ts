/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Snapshot } from '@postgres.ai/shared/types/api/entities/snapshot'

export const getEdgeSnapshots = (snapshots: Snapshot[]) => {
  const list = [...snapshots]
  const [first] = list
  const [last] = list.reverse()
  return {
    firstSnapshot: (first as Snapshot | undefined) ?? null,
    lastSnapshot: (last as Snapshot | undefined) ?? null
  }
}

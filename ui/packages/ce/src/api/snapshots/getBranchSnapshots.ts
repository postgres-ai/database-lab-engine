/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'
import {
  SnapshotDto,
  formatSnapshotDto,
} from '@postgres.ai/shared/types/api/entities/snapshot'
import { GetBranchSnapshots } from '@postgres.ai/shared/types/api/endpoints/getBranchSnapshots'

export const getBranchSnapshots: GetBranchSnapshots = async (
  branch: string,
) => {
  const response = await request(`/branch/snapshots/${branch}`)

  return {
    response: response.ok
      ? ((await response.json()) as SnapshotDto[]).map(formatSnapshotDto)
      : null,
    error: response.ok ? null : response,
  }
}

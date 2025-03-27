/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import {
  SnapshotDto,
  formatSnapshotDto,
} from '@postgres.ai/shared/types/api/entities/snapshot'
import { GetSnapshots } from '@postgres.ai/shared/types/api/endpoints/getSnapshots'

import { request } from 'helpers/request'

export const getSnapshots: GetSnapshots = async (req) => {
  const branchName = req.branchName?.trim()
  const action = branchName ? `/snapshots?branch=${branchName}` : '/snapshots'

  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: req.instanceId,
      method: 'get',
      action,
    }),
  })

  return {
    response: response.ok
      ? ((await response.json()) as SnapshotDto[]).map(formatSnapshotDto)
      : null,
    error: response.ok ? null : response,
  }
}

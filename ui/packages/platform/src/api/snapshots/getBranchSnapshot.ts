/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'
import { GetBranchSnapshot } from '@postgres.ai/shared/types/api/endpoints/getBranchSnapshot'

export const getBranchSnapshot: GetBranchSnapshot = async (
  snapshotId: string,
  instanceId: string,
) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: instanceId,
      action: `/branch/snapshot/${snapshotId}`,
      method: 'get',
    }),
  })

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}

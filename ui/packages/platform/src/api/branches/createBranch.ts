/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

import { CreateBranchFormValues } from '@postgres.ai/shared/types/api/endpoints/createBranch'

export const createBranch = async (req: CreateBranchFormValues) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: req.instanceId,
      action: '/branch',
      method: 'post',
      data: {
        branchName: req.branchName,
        ...(req.baseBranch && { baseBranch: req.baseBranch }),
        ...(req.snapshotID && { snapshotID: req.snapshotID }),
      },
    }),
  })

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}

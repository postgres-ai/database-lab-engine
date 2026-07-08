/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { UpdateBranch } from '@postgres.ai/shared/types/api/endpoints/updateBranch'

import { request } from 'helpers/request'

export const updateBranch: UpdateBranch = async (req) => {
  const response = await request(`/branch/${req.branchName}`, {
    method: 'PATCH',
    body: JSON.stringify({
      protected: req.branch.isProtected,
      ...(req.branch.protectionDurationMinutes !== undefined && {
        protectionDurationMinutes: req.branch.protectionDurationMinutes,
      }),
    }),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}

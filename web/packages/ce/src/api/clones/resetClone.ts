/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { ResetClone } from '@postgres.ai/shared/types/api/endpoints/resetClone'

import { request } from 'helpers/request'

export const resetClone: ResetClone = async (req) => {
  const response = await request(`/clone/${req.cloneId}/reset`, {
    method: 'POST',
    body: JSON.stringify({
      snapshotID: req.snapshotId,
    }),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}

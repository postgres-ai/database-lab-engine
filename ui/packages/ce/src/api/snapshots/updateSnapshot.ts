/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { UpdateSnapshot } from '@postgres.ai/shared/types/api/endpoints/updateSnapshot'

import { request } from 'helpers/request'

export const updateSnapshot: UpdateSnapshot = async (req) => {
  const response = await request(`/snapshot/${req.snapshotId}`, {
    method: 'PATCH',
    body: JSON.stringify({
      protected: req.snapshot.isProtected,
      ...(req.snapshot.protectionDurationMinutes !== undefined && {
        protectionDurationMinutes: req.snapshot.protectionDurationMinutes,
      }),
    }),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}

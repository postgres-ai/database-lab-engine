/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { UpdateClone } from '@postgres.ai/shared/types/api/endpoints/updateClone'

import { request } from 'helpers/request'

export const updateClone: UpdateClone = async (req) => {
  const response = await request(`/clone/${req.cloneId}`, {
    method: 'PATCH',
    body: JSON.stringify({
      protected: req.clone.isProtected,
    }),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}

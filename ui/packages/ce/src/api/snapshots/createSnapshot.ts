/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { CreateSnapshot } from '@postgres.ai/shared/types/api/endpoints/createSnapshot'

import { request } from 'helpers/request'

export const createSnapshot: CreateSnapshot = async (cloneId, message) => {
  const response = await request(`/branch/snapshot`, {
    method: 'POST',
    body: JSON.stringify({
      cloneID: cloneId,
      ...(message && { message: message }),
    }),
  })

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}

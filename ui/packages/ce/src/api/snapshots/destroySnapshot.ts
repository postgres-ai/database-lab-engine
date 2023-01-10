/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { DestroySnapshot } from '@postgres.ai/shared/types/api/endpoints/destroySnapshot'

import { request } from 'helpers/request'

export const destroySnapshot: DestroySnapshot = async (snapshotId) => {
  const response = await request(`/snapshot/delete`, {
    method: 'POST',
    body: JSON.stringify({
      snapshotID: snapshotId,
    }),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}

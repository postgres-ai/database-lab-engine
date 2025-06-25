/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

export const destroySnapshot = async (snapshotId: string, forceDelete: boolean) => {
  const response = await request(`/snapshot/${snapshotId}?force=${forceDelete}`, {
    method: 'DELETE'
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}

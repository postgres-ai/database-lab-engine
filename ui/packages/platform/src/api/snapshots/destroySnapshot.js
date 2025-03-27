/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

export const destroySnapshot = async (snapshotId, forceDelete, instanceId) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: instanceId,
      action: '/snapshot/delete',
      method: 'post',
      data: {
        snapshotID: snapshotId,
        force: forceDelete,
      },
    }),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}

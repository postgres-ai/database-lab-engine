/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

export const getSnapshotList = async (
  branchName: string,
  instanceId: string,
) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: instanceId,
      action: '/branch/log',
      method: 'post',
      data: {
        branchName: branchName,
      },
    }),
  })

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}

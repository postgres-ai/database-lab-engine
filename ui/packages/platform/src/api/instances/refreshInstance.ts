/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { RefreshInstance } from '@postgres.ai/shared/types/api/endpoints/refreshInstance'

import { request } from 'helpers/request'

export const refreshInstance: RefreshInstance = async (req) => {
  const response = await request('/rpc/dblab_instance_status_refresh', {
    method: 'post',
    body: JSON.stringify({
      instance_id: req.instanceId,
    }),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}

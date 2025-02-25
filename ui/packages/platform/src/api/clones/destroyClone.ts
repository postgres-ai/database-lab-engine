/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { DestroyClone } from '@postgres.ai/shared/types/api/endpoints/destroyClone'

import { request } from 'helpers/request'

export const destroyClone: DestroyClone = async (req) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      action: '/clone/' + encodeURIComponent(req.cloneId),
      instance_id: req.instanceId,
      method: 'delete'
    }),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}

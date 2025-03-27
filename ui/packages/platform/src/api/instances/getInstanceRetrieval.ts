/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2022, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'
import { formatInstanceRetrieval } from '@postgres.ai/shared/types/api/entities/instanceRetrieval'

interface GetInstanceRetrievalRequest {
  instanceId: string
}

export const getInstanceRetrieval = async (
  req: GetInstanceRetrievalRequest,
) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: req.instanceId,
      action: '/instance/retrieval',
      method: 'get',
    }),
  })

  return {
    response: response.ok
      ? formatInstanceRetrieval(await response.json())
      : null,
    error: response.ok ? null : response,
  }
}

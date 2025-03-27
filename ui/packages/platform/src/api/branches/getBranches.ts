/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'
import { formatBranchesDto } from '@postgres.ai/shared/types/api/endpoints/getBranches'

export const getBranches = async (instanceId: string) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: instanceId,
      action: '/branches',
      method: 'get',
    }),
  })

  return {
    response: response.ok ? formatBranchesDto(await response.json()) : null,
    error: response.ok ? null : response,
  }
}

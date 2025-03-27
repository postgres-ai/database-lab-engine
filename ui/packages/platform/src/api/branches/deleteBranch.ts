/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

export const deleteBranch = async (branchName: string, instanceId: string) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      action: `/branch/${branchName}`,
      instance_id: instanceId,
      method: 'delete'
    }),
  })

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : await response.json(),
  }
}

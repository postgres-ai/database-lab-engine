/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

export const deleteBranch = async (branchName: string) => {
  const response = await request(`/branch/${branchName}`, {
    method: 'DELETE'
  })

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : await response.json(),
  }
}

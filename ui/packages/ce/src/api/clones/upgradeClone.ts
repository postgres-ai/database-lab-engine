/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { UpgradeClone } from '@postgres.ai/shared/types/api/endpoints/upgradeClone'

import { request } from 'helpers/request'

export const upgradeClone: UpgradeClone = async (req) => {
  const response = await request(`/clone/${req.cloneId}/upgrade`, {
    method: 'POST',
    body: JSON.stringify(
      req.dockerImage ? { dockerImage: req.dockerImage } : {},
    ),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}

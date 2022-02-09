/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import {
  formatInstanceDto,
  InstanceDto,
} from '@postgres.ai/shared/types/api/entities/instance'
import { GetInstance } from '@postgres.ai/shared/types/api/endpoints/getInstance'

import { request } from 'helpers/request'

export const getInstance: GetInstance = async (req) => {
  const response = await request('/dblab_instances', {
    params: {
      id: `eq.${req.instanceId}`,
    },
  })

  return {
    response: response.ok
      ? ((await response.json()) as InstanceDto[]).map(formatInstanceDto)[0] ??
        null
      : null,
    error: response.ok ? null : response,
  }
}

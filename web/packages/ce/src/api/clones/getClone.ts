/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { GetClone } from '@postgres.ai/shared/types/api/endpoints/getClone'
import {
  CloneDto,
  formatCloneDto,
} from '@postgres.ai/shared/types/api/entities/clone'

import { request } from 'helpers/request'

export const getClone: GetClone = async (req) => {
  const response = await request(`/clone/${req.cloneId}`)

  return {
    response: response.ok
      ? formatCloneDto((await response.json()) as CloneDto)
      : null,
    error: response.ok ? null : response,
  }
}

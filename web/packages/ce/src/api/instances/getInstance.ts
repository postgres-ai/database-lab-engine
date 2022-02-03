/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { GetInstance } from '@postgres.ai/shared/types/api/endpoints/getInstance'
import { formatInstanceDto } from '@postgres.ai/shared/types/api/entities/instance'
import { InstanceStateDto } from '@postgres.ai/shared/types/api/entities/instanceState'

import { request } from 'helpers/request'

export const getInstance: GetInstance = async () => {
  const response = await request('/status')

  // Hack to get capability with platform API.
  const responseDto = response.ok
    ? {
        // Fake id which means nothing.
        id: 0,
        state: (await response.json()) as InstanceStateDto,
      }
    : null

  return {
    response: responseDto ? formatInstanceDto(responseDto) : null,
    error: response.ok ? null : response,
  }
}

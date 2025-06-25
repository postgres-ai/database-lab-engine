/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'
import { FullRefresh } from "@postgres.ai/shared/types/api/endpoints/fullRefresh";

export const fullRefresh: FullRefresh = async () => {
  const response = await request('/full-refresh', {
    method: "POST",
  })

  const result = response.ok ? await response.json() : null

  return {
    response: result,
    error: response.ok ? null : response,
  }
}

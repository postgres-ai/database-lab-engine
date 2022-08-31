/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2022, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

export const getInstanceRetrieval = async () => {
  const response = await request('/instance/retrieval')

  return {
    response: response.ok ? response : null,
    error: response.ok ? null : response,
  }
}

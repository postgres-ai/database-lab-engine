/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

export interface CloudRegion {
  api_name: string
  cloud_provider: string
  label: string
  native_code: string
  world_part: string
}

export const getCloudRegions = async (req: string) => {
  const response = await request(`/cloud_regions?cloud_provider=eq.${req}`)

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}

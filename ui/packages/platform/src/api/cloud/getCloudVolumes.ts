/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

export interface CloudVolumes {
  api_name: string
  type: string
  cloud_provider: string
  native_name: string
  native_reference_price_per_1000gib_per_hour: number
  native_reference_price_currency: string
  native_reference_price_region: string
  native_reference_price_revision_date: string
}

export const getCloudVolumes = async (cloud_provider: string) => {
  const response = await request(
    `/cloud_volumes?cloud_provider=eq.${cloud_provider}`,
  )

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}

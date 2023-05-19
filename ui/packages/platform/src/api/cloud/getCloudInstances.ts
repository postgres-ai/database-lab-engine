/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

export interface CloudInstance {
  api_name: string
  arch: string
  vcpus: number
  ram_gib: number
  dle_se_price_hourly: number
  cloud_provider: string
  only_in_regions: boolean | null
  native_name: string
  native_vcpus: number
  native_ram_gib: number
  native_reference_price_hourly: number
  native_reference_price_currency: string
  native_reference_price_region: string
  native_reference_price_revision_date: string
}

export interface CloudInstancesRequest {
  provider: string
  region: string
}

export const getCloudInstances = async (req: CloudInstancesRequest) => {
  const response = await request(
    `/cloud_instances?cloud_provider=eq.${req.provider}&only_in_regions&only_in_regions=ov.{all,${req.region}}&order=vcpus.asc,ram_gib.asc`,
  )

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}

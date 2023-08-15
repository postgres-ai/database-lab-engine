/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request'

export interface CloudImage {
  api_name: string
  os_name: string
  os_version: string
  arch: string
  cloud_provider: string
  region: string
  native_os_image: string
  release: string
}

export interface CloudImagesRequest {
  os_name: string
  os_version: string
  arch: string
  cloud_provider: string
  region: string
}

export const getCloudImages = async (req: CloudImagesRequest) => {
  const response = await request(
    `/cloud_os_images?os_name=eq.${req.os_name}&os_version=eq.${req.os_version}&arch=eq.${req.arch}&cloud_provider=eq.${req.cloud_provider}&region=eq.${req.region}`,
  )

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}

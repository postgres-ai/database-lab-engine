/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
import { capitalize } from '@postgres.ai/shared/utils/strings'

const STATUS_CODE_TO_TYPE = {
  OK: 'ok' as const,
  CREATING: 'waiting' as const,
  DELETING: 'waiting' as const,
  RESETTING: 'waiting' as const,
  FATAL: 'error' as const,
}

type StatusCode = keyof typeof STATUS_CODE_TO_TYPE

export const getCloneStatusType = (statusCode: StatusCode) =>
  STATUS_CODE_TO_TYPE[statusCode] ?? 'unknown'

export const getCloneStatusText = (statusCode: StatusCode) => {
  if (statusCode === 'OK') return statusCode

  return capitalize(statusCode)
}

export const checkIsCloneStable = (clone: Clone) =>
  clone.status.code === 'OK' || clone.status.code === 'FATAL'

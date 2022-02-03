/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { capitalize } from '@postgres.ai/shared/utils/strings'

const STATUS_CODE_TO_TYPE = {
  OK: 'ok' as const,
  WARNING: 'warning' as const,
  NO_RESPONSE: 'error' as const
}

export const getType = (code: keyof typeof STATUS_CODE_TO_TYPE) => {
  return STATUS_CODE_TO_TYPE[code] ?? 'unknown'
}

export const getText = (code: keyof typeof STATUS_CODE_TO_TYPE) => {
  if (code === 'OK') return code
  if (code === 'NO_RESPONSE') return 'No response'
  return capitalize(code)
}

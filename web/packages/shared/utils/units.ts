/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import byteSize from 'byte-size'

type Options = {
  precision?: number
}

export const formatBytesIEC = (bytes: number, options?: Options) => {
  const { precision = 3 } = options ?? {}

  const result = byteSize(bytes, {
    precision,
    units: 'iec',
  })

  return `${result.value} ${result.unit}`
}

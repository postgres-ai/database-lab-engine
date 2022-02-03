/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

type Options = {
  precision: number
  units: 'metric' | 'iec' | 'metric_octet' | 'iec_octet'
}

type Result = {
  value: string
  unit: string
  long: string
}

declare module 'byte-size' {
  declare const byteSize: (bytes: number, options?: Options) => Result
  export default byteSize
}

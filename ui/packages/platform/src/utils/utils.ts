/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

export const generateToken = function () {
  const a =
    'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890'.split('')
  const b = []

  for (let i = 0; i < 32; i++) {
    const j = (Math.random() * (a.length - 1)).toFixed(0)
    b[i] = a[j as keyof typeof a]
  }

  return b.join('')
}

export const isHttps = function (url: string | string[]) {
  return url && url.length > 5 && url.indexOf('https') === 0
}

export const snakeToCamel = (str: string) =>
  str.replace(/([-_]\w)/g, (g) => g[1].toUpperCase())

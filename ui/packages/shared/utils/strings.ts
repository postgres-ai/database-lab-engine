/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

export const capitalize = (value: string) => {
  const [firstChar, ...otherChars] = value.split('')
  return `${firstChar.toUpperCase()}${otherChars.join('').toLowerCase()}`
}

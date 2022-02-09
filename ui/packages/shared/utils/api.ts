/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

export const getTextFromUnknownApiError = async (error: Response) => {
  const log = (text: string) => console.error('Unknown API error', text)

  try {
    const result = await error.json()
    log(result)
    return JSON.stringify(result)
  } catch (e) {
    // not a json
  }

  try {
    const result = await error.text()
    log(result)
    return result
  } catch (e) {
    // not a text
  }

  const result = `${error.status} ${error.statusText}`
  log(result)
  return result
}

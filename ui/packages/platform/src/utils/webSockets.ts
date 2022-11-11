/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

export const createWebSocket = function (url: string, path: string) {
  // Here expected that url started with ws://
  const wsUri = url + '/' + path

  return new WebSocket(wsUri)
}

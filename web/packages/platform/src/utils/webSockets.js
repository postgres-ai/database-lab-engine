/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

module.exports = {
  createWebSocket: function (url, path) {
    // Here expected that url started with ws://
    let wsUri = url + '/' + path;

    return new WebSocket(wsUri);
  }
};

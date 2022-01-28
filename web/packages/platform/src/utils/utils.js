/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

module.exports = {
  generateToken: function () {
    let a =
      'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890'.split(
        '');
    let b = [];

    for (let i = 0; i < 32; i++) {
      let j = (Math.random() * (a.length - 1)).toFixed(0);
      b[i] = a[j];
    }

    return b.join('');
  },

  isHttps: function (url) {
    return url && url.length > 5 && url.indexOf('https') === 0;
  },

  snakeToCamel: str => str.replace(/([-_]\w)/g, g => g[1].toUpperCase())
};

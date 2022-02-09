/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import 'es6-promise/auto';
import 'whatwg-fetch';

class ExplainPev2Api {
  constructor(setting) {
    this.server = setting.explainPev2Server;
  }

  post(url, data, options = {}) {
    let fetchOptions = {
      ...options,
      method: 'post',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    };

    return fetch(url, fetchOptions);
  }

  postPlan(plan, query) {
    return this.post(`${this.server}/api/rpc/post_plan`, {
      plan: plan,
      query: query || ''
    });
  }
}

export default ExplainPev2Api;

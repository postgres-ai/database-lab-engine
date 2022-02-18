/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import 'es6-promise/auto';
import 'whatwg-fetch';

class ExplainDepeszApi {
  constructor(setting) {
    this.server = setting.explainDepeszServer;
  }

  post(url, data, options = {}) {
    let fetchOptions = {
      ...options,
      method: 'post',
      body: data
    };

    return fetch(url, fetchOptions);
  }

  postPlan(plan, query) {
    const formData = new FormData();
    formData.append('is_public', '0');
    formData.append('plan', plan);
    formData.append('query', query);

    return this.post(this.server, formData);
  }
}

export default ExplainDepeszApi;

/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

const API_SERVER = process.env.REACT_APP_API_SERVER;
const WS_SERVER = process.env.REACT_APP_WS_SERVER;
const SIGNIN_URL = process.env.REACT_APP_SIGNIN_URL;
const AUTH_URL = process.env.REACT_APP_AUTH_URL;
const ROOT_URL = process.env.REACT_APP_ROOT_URL;
const EXPLAIN_DEPESZ_SERVER = process.env.REACT_APP_EXPLAIN_DEPESZ_SERVER;
const EXPLAIN_PEV2_SERVER = process.env.REACT_APP_EXPLAIN_PEV2_SERVER;
const STRIPE_API_KEY = process.env.REACT_APP_STRIPE_API_KEY;

const settings = {
  server: window.location.protocol + '//' + window.location.host,
  apiServer: API_SERVER,
  wsServer: WS_SERVER,
  signinUrl: SIGNIN_URL,
  authUrl: AUTH_URL,
  rootUrl: ROOT_URL,
  explainDepeszServer: EXPLAIN_DEPESZ_SERVER,
  explainPev2Server: EXPLAIN_PEV2_SERVER,
  stripeApiKey: STRIPE_API_KEY,
  demoOrgAlias: 'demo',
  init: function (cb: () => void) {
    this.server = window.location.protocol + '//' + window.location.host;
    this.apiServer = API_SERVER;
    this.wsServer = WS_SERVER;
    this.signinUrl = SIGNIN_URL;
    this.authUrl = AUTH_URL;
    this.explainDepeszServer = EXPLAIN_DEPESZ_SERVER;
    this.stripeApiKey = STRIPE_API_KEY;
    if (cb) {
      cb();
    }
  }
};

export default settings;

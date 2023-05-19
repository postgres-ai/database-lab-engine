/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

export const NODE_ENV = process.env.NODE_ENV
export const SENTRY_DSN = process.env.REACT_APP_SENTRY_DSN
export const API_URL_PREFIX = 'https://postgres.ai/api/general' ?? ''
export const WS_URL_PREFIX = process.env.REACT_APP_WS_URL_PREFIX ?? ''
export const BUILD_TIMESTAMP = process.env.BUILD_TIMESTAMP

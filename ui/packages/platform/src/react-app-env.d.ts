/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

// eslint-disable-next-line
/// <reference types="react-scripts" />

// Env types.
declare namespace NodeJS {
  interface ProcessEnv {
    readonly REACT_APP_SENTRY_DSN: string | undefined
    readonly REACT_APP_API_SERVER: string | undefined
    readonly BUILD_TIMESTAMP: number
  }
}

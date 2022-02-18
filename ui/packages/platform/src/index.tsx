/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import ReactDOM from 'react-dom'
import * as Sentry from '@sentry/react'
import { Integrations } from '@sentry/tracing'

import { initConfig } from '@postgres.ai/shared/config'

import { NODE_ENV, SENTRY_DSN } from 'config/env'

import App from './App'
import { unregister } from './registerServiceWorker'

import './index.scss'

if (NODE_ENV === 'production' && SENTRY_DSN) {
  Sentry.init({
    dsn: SENTRY_DSN,
    integrations: [new Integrations.BrowserTracing()],
    environment: NODE_ENV,
    // We recommend adjusting this value in production, or using tracesSampler
    // for finer control
    tracesSampleRate: 1.0,
  })
}

const main = async () => {
  await initConfig()
  ReactDOM.render(<App />, document.getElementById('root'))
}

main()

// This func disable service worker.
// Don't remove it,
// because we cannot be sure that all previous users uninstalled their service workers.
// It should be permanent, except when you want to add new service worker.
const disableSw = () => {
  unregister()

  // clear all sw caches
  if ('caches' in window) {
    window.caches.keys().then((res) => {
      res.forEach((key) => {
        window.caches.delete(key)
      })
    })
  }
}

window.addEventListener('load', disableSw)

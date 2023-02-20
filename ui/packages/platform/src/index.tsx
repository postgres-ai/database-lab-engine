/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import ReactDOM from 'react-dom'

import { initConfig } from '@postgres.ai/shared/config'

import App from './App'
import { unregister } from './registerServiceWorker'

import './index.scss'

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

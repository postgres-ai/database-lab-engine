/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react'
import { BrowserRouter as Router, Route } from 'react-router-dom'

import { ROUTES } from 'config/routes'

import IndexPage from './components/IndexPage'

class App extends Component {
  render() {
    return (
      <Router basename={process.env.PUBLIC_URL}>
        <Route path={ROUTES.ROOT.path} component={IndexPage} />
      </Router>
    )
  }
}

export default App

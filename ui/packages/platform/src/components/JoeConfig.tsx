/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'

import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

class JoeConfig extends Component {
  render() {
    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        {...this.props}
        breadcrumbs={[{ name: 'SQL Optimization' }, { name: 'Configuration' }]}
      />
    )

    return (
      <div>
        {breadcrumbs}
        <br />
        See configuration guides in&nbsp;
        <a href="https://gitlab.com/postgres-ai/joe" target="blank">
          postgres-ai/joe
        </a>
        &nbsp;repository.
      </div>
    )
  }
}

export default JoeConfig

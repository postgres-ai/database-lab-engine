import { Switch, Route, Redirect } from 'react-router-dom'

import { ROUTES } from 'config/routes'

import { Logs } from './Logs'
import { Page } from './Page'
import { Clones } from './Clones'
import { Branches } from './Branches'
import { Snapshots } from './Snapshots'
import { Configuration } from './Configuration'

export const Instance = () => {
  return (
    <Switch>
      <Route exact path={ROUTES.INSTANCE.path}>
        <Page />
      </Route>
      <Route path={ROUTES.INSTANCE.CLONES.path}>
        <Clones />
      </Route>
      <Route path={ROUTES.INSTANCE.SNAPSHOTS.path}>
        <Snapshots />
      </Route>
      <Route path={ROUTES.INSTANCE.BRANCHES.path}>
        <Branches />
      </Route>
      <Route path={ROUTES.INSTANCE.LOGS.path}>
        <Logs />
      </Route>
      <Route path={ROUTES.INSTANCE.CONFIGURATION.path}>
        <Configuration />
      </Route>
      <Redirect to={ROUTES.path} />
    </Switch>
  )
}

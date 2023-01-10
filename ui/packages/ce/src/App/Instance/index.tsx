import { Switch, Route, Redirect } from 'react-router-dom'

import { ROUTES } from 'config/routes'

import { Page } from './Page'
import { Clones } from './Clones'
import { Snapshots } from './Snapshots'
import { Branches } from './Branches'

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
      <Redirect to={ROUTES.path} />
    </Switch>
  )
}

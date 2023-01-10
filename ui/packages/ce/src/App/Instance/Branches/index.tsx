import { Switch, Route, Redirect } from 'react-router-dom'

import { ROUTES } from 'config/routes'
import { TABS_INDEX } from '@postgres.ai/shared/pages/Instance/Tabs'

import { Page } from '../Page'
import { Branch } from './Branch'

export const Branches = () => {
  return (
    <Switch>
      <Route exact path={ROUTES.INSTANCE.BRANCHES.BRANCHES.path}>
        <Page renderCurrentTab={TABS_INDEX.BRANCHES} />
      </Route>
      <Route exact path={ROUTES.INSTANCE.BRANCHES.BRANCH.createPath()}>
        <Branch />
      </Route>
      <Redirect to={ROUTES.path} />
    </Switch>
  )
}

import { Switch, Route, Redirect } from 'react-router-dom'

import { TABS_INDEX } from '@postgres.ai/shared/pages/Instance/Tabs'

import { ROUTES } from 'config/routes'

import { Page } from '../Page'
import { Snapshot } from './Snapshot'

export const Snapshots = () => {
  return (
    <Switch>
      <Route exact path={ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOTS.path}>
        <Page renderCurrentTab={TABS_INDEX.SNAPSHOTS} />
      </Route>
      <Route exact path={ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOT.createPath()}>
        <Snapshot />
      </Route>
      <Redirect to={ROUTES.path} />
    </Switch>
  )
}

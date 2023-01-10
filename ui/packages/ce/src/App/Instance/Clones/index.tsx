import { Switch, Route, Redirect } from 'react-router-dom'

import { TABS_INDEX } from '@postgres.ai/shared/pages/Instance/Tabs'

import { ROUTES } from 'config/routes'

import { CreateClone } from './CreateClone'
import { Clone } from './Clone'
import { Page } from '../Page'

export const Clones = () => {
  return (
    <Switch>
      <Route exact path={ROUTES.INSTANCE.CLONES.CREATE.path}>
        <CreateClone />
      </Route>

      <Route exact path={ROUTES.INSTANCE.CLONES.CLONE.createPath()}>
        <Clone />
      </Route>

      <Route exact path={ROUTES.INSTANCE.CLONES.CLONES.path}>
        <Page renderCurrentTab={TABS_INDEX.CLONES} />
      </Route>

      <Redirect to={ROUTES.path} />
    </Switch>
  )
}

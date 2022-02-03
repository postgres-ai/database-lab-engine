import { Switch, Route, Redirect } from 'react-router-dom'

import { ROUTES } from 'config/routes'

import { Page } from './Page'
import { Clones } from './Clones'

export const Instance = () => {
  return (
    <Switch>
      <Route exact path={ROUTES.INSTANCE.path}>
        <Page />
      </Route>
      <Route path={ROUTES.INSTANCE.CLONES.path}>
        <Clones />
      </Route>
      <Redirect to={ROUTES.path} />
    </Switch>
  )
}

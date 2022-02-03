import { Switch, Route, Redirect } from 'react-router-dom'

import { ROUTES } from 'config/routes'

import { CreateClone } from './CreateClone'
import { Clone } from './Clone'

export const Clones = () => {
  return (
    <Switch>
      <Route exact path={ROUTES.INSTANCE.CLONES.CREATE.path}>
        <CreateClone />
      </Route>

      <Route exact path={ROUTES.INSTANCE.CLONES.CLONE.createPath()}>
        <Clone />
      </Route>

      <Redirect to={ROUTES.path} />
    </Switch>
  )
}

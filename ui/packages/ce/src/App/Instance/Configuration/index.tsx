import { TABS_INDEX } from '@postgres.ai/shared/pages/Instance/Tabs'
import { ROUTES } from 'config/routes'
import { Route } from 'react-router'
import { Page } from '../Page'

export const Configuration = () => (
  <Route exact path={ROUTES.INSTANCE.CONFIGURATION.path}>
    <Page renderCurrentTab={TABS_INDEX.CONFIGURATION} />
  </Route>
)

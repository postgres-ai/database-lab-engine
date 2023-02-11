import { useEffect } from 'react'
import { observer } from 'mobx-react-lite'
import { BrowserRouter, Switch, Route, Redirect } from 'react-router-dom'

import { StubSpinner } from '@postgres.ai/shared/components/StubSpinnerFlex'

import { appStore } from 'stores/app'
import { ROUTES } from 'config/routes'

import { Layout } from './Layout'
import { Menu } from './Menu'
import { Instance } from './Instance'
import { Auth } from './Auth'

export const App = observer(() => {
  useEffect(() => {
    appStore.loadData()
  }, [])

  if (appStore.engine.isLoading || appStore.engine.data === undefined)
    return <StubSpinner />

  return (
    <BrowserRouter>
      <Layout menu={<Menu isValidToken={appStore.isValidAuthToken} />}>
        {appStore.isValidAuthToken ? (
          <Switch>
            <Route path={ROUTES.INSTANCE.path}>
              <Instance />
            </Route>
            <Redirect to={ROUTES.INSTANCE.path} />
          </Switch>
        ) : (
          <Switch>
            <Route path={ROUTES.AUTH.path}>
              <Auth />
            </Route>
            <Redirect to={ROUTES.AUTH.path} />
          </Switch>
        )}
      </Layout>
    </BrowserRouter>
  )
})

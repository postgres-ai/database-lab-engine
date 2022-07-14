import { Instance } from '@postgres.ai/shared/pages/Instance'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'
import { getInstance } from 'api/instances/getInstance'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { destroyClone } from 'api/clones/destroyClone'
import { resetClone } from 'api/clones/resetClone'
import { getWSToken } from "api/engine/getWSToken";
import { initWS } from "api/engine/initWS";

export const Page = () => {
  const routes = {
    createClone: () => ROUTES.INSTANCE.CLONES.CREATE.path,
    clone: (cloneId: string) =>
      ROUTES.INSTANCE.CLONES.CLONE.createPath(cloneId),
  }

  const api = {
    getInstance,
    getSnapshots,
    destroyClone,
    resetClone,
    getWSToken,
    initWS,
  }

  const elements = {
    breadcrumbs: <NavPath routes={[ROUTES, ROUTES.INSTANCE]} />,
  }

  return (
    <PageContainer>
      <Instance
        title={'Instance'}
        instanceId={''}
        routes={routes}
        api={api}
        elements={elements}
      />
    </PageContainer>
  )
}

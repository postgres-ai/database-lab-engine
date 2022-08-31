import { Instance } from '@postgres.ai/shared/pages/Instance'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'
import { getInstance } from 'api/instances/getInstance'
import { getInstanceRetrieval } from 'api/instances/getInstanceRetrieval'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { destroyClone } from 'api/clones/destroyClone'
import { resetClone } from 'api/clones/resetClone'
import { getWSToken } from "api/engine/getWSToken";
import { initWS } from "api/engine/initWS";
import { getConfig } from 'api/configs/getConfig'
import { getFullConfig } from 'api/configs/getFullConfig'
import { updateConfig } from 'api/configs/updateConfig'
import { testDbSource } from 'api/configs/testDbSource'

export const Page = () => {
  const routes = {
    createClone: () => ROUTES.INSTANCE.CLONES.CREATE.path,
    clone: (cloneId: string) =>
      ROUTES.INSTANCE.CLONES.CLONE.createPath(cloneId),
  }

  const api = {
    getInstance,
    getInstanceRetrieval,
    getSnapshots,
    destroyClone,
    resetClone,
    getWSToken,
    getConfig,
    getFullConfig,
    updateConfig,
    testDbSource,
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

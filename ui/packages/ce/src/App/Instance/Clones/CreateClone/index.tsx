import { CreateClone as CreateClonePage } from '@postgres.ai/shared/pages/CreateClone'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'
import { getInstance } from 'api/instances/getInstance'
import { getInstanceRetrieval } from 'api/instances/getInstanceRetrieval'
import { createClone } from 'api/clones/createClone'
import { getClone } from 'api/clones/getClone'
import { getBranches } from 'api/branches/getBranches'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { initWS } from 'api/engine/initWS'

export const CreateClone = () => {
  const routes = {
    clone: (cloneId: string) =>
      ROUTES.INSTANCE.CLONES.CLONE.createPath(cloneId),
  }

  const api = {
    getInstance,
    getInstanceRetrieval,
    createClone,
    getClone,
    getBranches,
    getSnapshots,
    initWS
  }

  const elements = {
    breadcrumbs: (
      <NavPath
        routes={[
          ROUTES,
          ROUTES.INSTANCE.CLONES.CLONES,
          ROUTES.INSTANCE.CLONES.CREATE,
        ]}
      />
    ),
  }

  return (
    <PageContainer>
      <CreateClonePage
        instanceId={''}
        routes={routes}
        api={api}
        elements={elements}
      />
    </PageContainer>
  )
}

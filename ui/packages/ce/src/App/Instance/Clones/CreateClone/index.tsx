import { CreateClone as CreateClonePage } from '@postgres.ai/shared/pages/CreateClone'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'
import { getInstance } from 'api/instances/getInstance'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { createClone } from 'api/clones/createClone'
import { getClone } from 'api/clones/getClone'

export const CreateClone = () => {
  const routes = {
    clone: (cloneId: string) =>
      ROUTES.INSTANCE.CLONES.CLONE.createPath(cloneId),
  }

  const api = {
    getSnapshots,
    getInstance,
    createClone,
    getClone,
  }

  const elements = {
    breadcrumbs: (
      <NavPath
        routes={[ROUTES, ROUTES.INSTANCE, ROUTES.INSTANCE.CLONES.CREATE]}
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

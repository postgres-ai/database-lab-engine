import { useParams } from 'react-router-dom'

import { Clone as ClonePage } from '@postgres.ai/shared/pages/Clone'

import { getSnapshots } from 'api/snapshots/getSnapshots'
import { getInstance } from 'api/instances/getInstance'
import { getClone } from 'api/clones/getClone'
import { resetClone } from 'api/clones/resetClone'
import { destroyClone } from 'api/clones/destroyClone'
import { updateClone } from 'api/clones/updateClone'
import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'

type Params = {
  cloneId: string
}

export const Clone = () => {
  const { cloneId } = useParams<Params>()

  const api = {
    getSnapshots,
    getInstance,
    getClone,
    resetClone,
    destroyClone,
    updateClone,
  }

  const elements = {
    breadcrumbs: (
      <NavPath
        routes={[
          ROUTES,
          ROUTES.INSTANCE,
          {
            name: ROUTES.INSTANCE.CLONES.CLONE.name,
            path: ROUTES.INSTANCE.CLONES.CLONE.createPath(cloneId),
          },
        ]}
      />
    ),
  }

  return (
    <PageContainer>
      <ClonePage
        instanceId={''}
        cloneId={cloneId}
        routes={{
          instance: () => ROUTES.INSTANCE.path,
        }}
        api={api}
        elements={elements}
      />
    </PageContainer>
  )
}

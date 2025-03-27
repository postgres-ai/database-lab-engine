import { useParams } from 'react-router-dom'

import { Clone as ClonePage } from '@postgres.ai/shared/pages/Clone'

import { getSnapshots } from 'api/snapshots/getSnapshots'
import { getInstance } from 'api/instances/getInstance'
import { getInstanceRetrieval } from 'api/instances/getInstanceRetrieval'
import { getClone } from 'api/clones/getClone'
import { resetClone } from 'api/clones/resetClone'
import { destroyClone } from 'api/clones/destroyClone'
import { updateClone } from 'api/clones/updateClone'
import { createSnapshot } from 'api/snapshots/createSnapshot'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'
import { destroySnapshot } from 'api/snapshots/destroySnapshot'

type Params = {
  cloneId: string
}

export const Clone = () => {
  const { cloneId } = useParams<Params>()

  const api = {
    getSnapshots,
    getInstance,
    getInstanceRetrieval,
    getClone,
    resetClone,
    destroyClone,
    destroySnapshot,
    updateClone,
    createSnapshot,
  }

  const elements = {
    breadcrumbs: (
      <NavPath
        routes={[
          ROUTES,
          ROUTES.INSTANCE.CLONES.CLONES,
          {
            name: `${ROUTES.INSTANCE.CLONES.CLONE.name}/${cloneId}`,
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
          snapshot: (snapshotId: string) =>
            ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOT.createPath(snapshotId),
        }}
        api={api}
        elements={elements}
      />
    </PageContainer>
  )
}

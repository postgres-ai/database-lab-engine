import { useParams } from 'react-router-dom'

import { SnapshotPage } from '@postgres.ai/shared/pages/Snapshots/Snapshot'

import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'
import { PageContainer } from 'components/PageContainer'

import { destroySnapshot } from 'api/snapshots/destroySnapshot'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { getBranchSnapshot } from 'api/snapshots/getBranchSnapshot'

type Params = {
  snapshotId: string
}

export const Snapshot = () => {
  const { snapshotId } = useParams<Params>()

  const api = {
    destroySnapshot,
    getSnapshots,
    getBranchSnapshot,
  }

  const elements = {
    breadcrumbs: (
      <NavPath
        routes={[
          ROUTES,
          ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOTS,
          {
            name: `${ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOT.name}/${snapshotId}`,
            path: ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOT.createPath(snapshotId),
          },
        ]}
      />
    ),
  }

  return (
    <PageContainer>
      <SnapshotPage
        instanceId={''}
        snapshotId={snapshotId}
        routes={{
          snapshot: () => ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOTS.path,
        }}
        api={api}
        elements={elements}
      />
    </PageContainer>
  )
}

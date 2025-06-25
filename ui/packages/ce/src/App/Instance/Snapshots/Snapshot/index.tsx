import { useParams } from 'react-router-dom'

import { SnapshotPage } from '@postgres.ai/shared/pages/Snapshots/Snapshot'

import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'
import { PageContainer } from 'components/PageContainer'

import { destroySnapshot } from 'api/snapshots/destroySnapshot'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { getBranchSnapshot } from 'api/snapshots/getBranchSnapshot'
import { initWS } from 'api/engine/initWS'

type Params = {
  snapshotId: string
}

export const Snapshot = () => {
  const { snapshotId } = useParams<Params>()

  const api = {
    destroySnapshot,
    getSnapshots,
    getBranchSnapshot,
    initWS,
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
          snapshots: () => ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOTS.path,
          snapshot: () => ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOTS.path,
          branch: (branchName: string) =>
            ROUTES.INSTANCE.BRANCHES.BRANCH.createPath(branchName),
          clone: (cloneId: string) =>
            ROUTES.INSTANCE.CLONES.CLONE.createPath(cloneId),
          createClone: (branchId: string, snapshotId: string) => ROUTES.INSTANCE.CLONES.CREATE.createPath(branchId, snapshotId),
        }}
        api={api}
        elements={elements}
      />
    </PageContainer>
  )
}

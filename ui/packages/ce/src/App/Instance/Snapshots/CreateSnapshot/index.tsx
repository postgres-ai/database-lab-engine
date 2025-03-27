import { createSnapshot } from 'api/snapshots/createSnapshot'
import { getInstance } from 'api/instances/getInstance'

import { CreateSnapshotPage } from '@postgres.ai/shared/pages/CreateSnapshot'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'

export const CreateSnapshot = () => {
  const api = {
    createSnapshot,
    getInstance,
  }

  const elements = {
    breadcrumbs: (
      <NavPath
        routes={[
          ROUTES,
          ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOTS,
          ROUTES.INSTANCE.SNAPSHOTS.CREATE,
        ]}
      />
    ),
  }

  return (
    <PageContainer>
      <CreateSnapshotPage
        instanceId=""
        api={api}
        elements={elements}
        routes={{
          snapshot: (snapshotId: string) =>
            ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOT.createPath(snapshotId),
        }}
      />
    </PageContainer>
  )
}

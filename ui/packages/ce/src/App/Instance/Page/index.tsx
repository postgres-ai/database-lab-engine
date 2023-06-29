
import { Instance } from '@postgres.ai/shared/pages/Instance'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'
import { getInstance } from 'api/instances/getInstance'
import { getInstanceRetrieval } from 'api/instances/getInstanceRetrieval'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { createSnapshot } from 'api/snapshots/createSnapshot'
import { destroyClone } from 'api/clones/destroyClone'
import { resetClone } from 'api/clones/resetClone'
import { getWSToken } from 'api/engine/getWSToken'
import { initWS } from 'api/engine/initWS'
import { getConfig } from 'api/configs/getConfig'
import { getFullConfig } from 'api/configs/getFullConfig'
import { updateConfig } from 'api/configs/updateConfig'
import { testDbSource } from 'api/configs/testDbSource'
import { getEngine } from 'api/engine/getEngine'
import { createBranch } from 'api/branches/createBranch'
import { getBranches } from 'api/branches/getBranches'
import { getSnapshotList } from 'api/branches/getSnapshotList'
import { deleteBranch } from 'api/branches/deleteBranch'

export const Page = ({ renderCurrentTab }: { renderCurrentTab?: number }) => {
  const routes = {
    createClone: () => ROUTES.INSTANCE.CLONES.CREATE.path,
    createBranch: () => ROUTES.INSTANCE.BRANCHES.CREATE.path,
    createSnapshot: () => ROUTES.INSTANCE.SNAPSHOTS.CREATE.path,
    clone: (cloneId: string) =>
      ROUTES.INSTANCE.CLONES.CLONE.createPath(cloneId),
  }

  const api = {
    getInstance,
    getInstanceRetrieval,
    getSnapshots,
    createSnapshot,
    destroyClone,
    resetClone,
    getWSToken,
    getConfig,
    getFullConfig,
    updateConfig,
    testDbSource,
    initWS,
    getEngine,
    createBranch,
    getBranches,
    getSnapshotList,
    deleteBranch
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
        renderCurrentTab={renderCurrentTab}
      />
    </PageContainer>
  )
}

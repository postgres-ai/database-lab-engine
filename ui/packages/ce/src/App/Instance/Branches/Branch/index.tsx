import { useParams } from 'react-router-dom'

import { getBranches } from 'api/branches/getBranches'
import { deleteBranch } from 'api/branches/deleteBranch'
import { getSnapshotList } from 'api/branches/getSnapshotList'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'
import { BranchesPage } from '@postgres.ai/shared/pages/Branches/Branch'

type Params = {
  branchId: string
}

export const Branch = () => {
  const { branchId } = useParams<Params>()

  const api = {
    getBranches,
    deleteBranch,
    getSnapshotList,
  }

  const elements = {
    breadcrumbs: (
      <NavPath
        routes={[
          ROUTES,
          ROUTES.INSTANCE.BRANCHES.BRANCHES,
          {
            name: `${ROUTES.INSTANCE.BRANCHES.BRANCH.name}/${branchId}`,
            path: ROUTES.INSTANCE.BRANCHES.BRANCH.createPath(branchId),
          },
        ]}
      />
    ),
  }

  return (
    <PageContainer>
      <BranchesPage
        instanceId=""
        api={api}
        elements={elements}
        branchId={branchId}
        routes={{
          branch: () => ROUTES.INSTANCE.BRANCHES.BRANCHES.path,
          branches: () => ROUTES.INSTANCE.BRANCHES.BRANCHES.path,
          snapshot: (snapshotId: string) =>
            ROUTES.INSTANCE.SNAPSHOTS.SNAPSHOT.createPath(snapshotId),
          createClone: () => ROUTES.INSTANCE.CLONES.CREATE.path,
        }}
      />
    </PageContainer>
  )
}

import { getBranches } from 'api/branches/getBranches'
import { createBranch } from 'api/branches/createBranch'
import { getSnapshotList } from 'api/branches/getSnapshotList'

import { CreateBranchPage } from '@postgres.ai/shared/pages/CreateBranch'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'

export const CreateBranch = () => {
  const api = {
    getBranches,
    createBranch,
    getSnapshotList,
  }

  const elements = {
    breadcrumbs: (
      <NavPath
        routes={[
          ROUTES,
          ROUTES.INSTANCE.BRANCHES.BRANCHES,
          ROUTES.INSTANCE.BRANCHES.CREATE,
        ]}
      />
    ),
  }

  return (
    <PageContainer>
      <CreateBranchPage api={api} elements={elements} />
    </PageContainer>
  )
}

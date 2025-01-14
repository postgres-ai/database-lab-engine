import { getBranches } from 'api/branches/getBranches'
import { createBranch } from 'api/branches/createBranch'
import { getBranchSnapshots } from 'api/snapshots/getBranchSnapshots'

import { CreateBranchPage } from '@postgres.ai/shared/pages/CreateBranch'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'

export const CreateBranch = () => {
  const api = {
    getBranches,
    createBranch,
    getBranchSnapshots,
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

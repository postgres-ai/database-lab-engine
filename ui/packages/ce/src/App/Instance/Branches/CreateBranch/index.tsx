import { getBranches } from 'api/branches/getBranches'
import { createBranch } from 'api/branches/createBranch'
import { getSnapshots } from 'api/snapshots/getSnapshots'

import { CreateBranchPage } from '@postgres.ai/shared/pages/CreateBranch'

import { PageContainer } from 'components/PageContainer'
import { NavPath } from 'components/NavPath'
import { ROUTES } from 'config/routes'

export const CreateBranch = () => {
  const api = {
    getBranches,
    createBranch,
    getSnapshots,
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
      <CreateBranchPage instanceId={''} api={api} elements={elements} />
    </PageContainer>
  )
}

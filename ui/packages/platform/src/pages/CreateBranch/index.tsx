import { useParams } from 'react-router-dom'

import { CreateBranchPage } from '@postgres.ai/shared/pages/CreateBranch'

import { getBranches } from 'api/branches/getBranches'
import { createBranch } from 'api/branches/createBranch'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import { ROUTES } from 'config/routes'

type Params = {
  org: string
  project?: string
  instanceId: string
  branchId: string
}

export const CreateBranch = () => {
  const params = useParams<Params>()

  const routes = {
    branch: (branchId: string) =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.BRANCHES.BRANCH.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
            branchId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.BRANCHES.BRANCH.createPath({
            ...params,
            branchId,
          }),
  }

  const api = {
    getBranches,
    createBranch,
    getSnapshots,
  }

  const elements = {
    breadcrumbs: (
      <ConsoleBreadcrumbsWrapper
        org={params.org}
        project={params.project}
        hasDivider
        breadcrumbs={[
          { name: 'Database Lab Instances', url: 'instances' },
          { name: 'Instance #' + params.instanceId, url: params.instanceId },
          {
            name: 'Branches',
            url: 'branches',
          },
          {
            name: 'Create Branch ',
            url: null,
          },
        ]}
      />
    ),
  }

  return (
    <CreateBranchPage
      instanceId={params.instanceId}
      routes={routes}
      api={api}
      elements={elements}
    />
  )
}

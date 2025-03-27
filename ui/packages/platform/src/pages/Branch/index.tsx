import { useParams } from 'react-router-dom'

import { BranchesPage } from '@postgres.ai/shared/pages/Branches/Branch'

import { getBranches } from 'api/branches/getBranches'
import { deleteBranch } from 'api/branches/deleteBranch'
import { getSnapshotList } from 'api/branches/getSnapshotList'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import { ROUTES } from 'config/routes'

type Params = {
  org: string
  project?: string
  instanceId: string
  branchId: string
}

export const Branch = () => {
  const params = useParams<Params>()

  const routes = {
    branch: () =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.createPath(params),
    branches: () =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.BRANCHES.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.BRANCHES.createPath({
            org: params.org,
            instanceId: params.instanceId,
          }),
    snapshot: (snapshotId: string) =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.SNAPSHOTS.SNAPSHOT.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
            snapshotId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.SNAPSHOTS.SNAPSHOT.createPath({
            org: params.org,
            instanceId: params.instanceId,
            snapshotId,
          }),
  }

  const api = {
    getBranches,
    deleteBranch,
    getSnapshotList,
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
            name: 'Branch ' + params.branchId,
            url: null,
          },
        ]}
      />
    ),
  }

  return (
    <BranchesPage
      branchId={params.branchId}
      instanceId={params.instanceId}
      routes={routes}
      api={api}
      elements={elements}
    />
  )
}

import { useParams } from 'react-router-dom'

import { SnapshotPage } from '@postgres.ai/shared/pages/Snapshots/Snapshot'

import { getSnapshots } from 'api/snapshots/getSnapshots'
import { destroySnapshot } from 'api/snapshots/destroySnapshot'
import { getBranchSnapshot } from 'api/snapshots/getBranchSnapshot'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import { ROUTES } from 'config/routes'

type Params = {
  org: string
  project?: string
  instanceId: string
  snapshotId: string
}

export const Snapshot = () => {
  const params = useParams<Params>()

  const routes = {
    snapshot: () =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.createPath(params),
    snapshots: () =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.SNAPSHOTS.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.SNAPSHOTS.createPath(params),
    branch: (branchName: string) =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.BRANCHES.BRANCH.createPath({
            org: params.org,
            project: params.project,
            branchId: branchName,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.BRANCHES.BRANCH.createPath({
            org: params.org,
            branchId: branchName,
            instanceId: params.instanceId,
          }),
    clone: (cloneId: string) =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.CLONES.CLONE.createPath({
            org: params.org,
            project: params.project,
            cloneId: cloneId,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.CLONES.CLONE.createPath({
            org: params.org,
            cloneId: cloneId,
            instanceId: params.instanceId,
          }),
  }

  const api = {
    destroySnapshot,
    getSnapshots,
    getBranchSnapshot,
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
            name: 'Snapshots',
            url: 'snapshots',
          },
          {
            name: 'Snapshot ' + params.snapshotId,
            url: null,
          },
        ]}
      />
    ),
  }

  return (
    <SnapshotPage
      snapshotId={params.snapshotId}
      instanceId={params.instanceId}
      routes={routes}
      api={api}
      elements={elements}
    />
  )
}

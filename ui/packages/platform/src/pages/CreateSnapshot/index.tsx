import { useParams } from 'react-router-dom'

import { CreateSnapshotPage } from '@postgres.ai/shared/pages/CreateSnapshot'

import { getInstance } from 'api/instances/getInstance'
import { createSnapshot } from 'api/snapshots/createSnapshot'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import { ROUTES } from 'config/routes'

type Params = {
  org: string
  project?: string
  instanceId: string
  snapshotId: string
}

export const CreateSnapshot = () => {
  const params = useParams<Params>()

  const routes = {
    snapshot: (snapshotId: string) =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.SNAPSHOTS.SNAPSHOT.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
            snapshotId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.SNAPSHOTS.SNAPSHOT.createPath({
            ...params,
            snapshotId,
          }),
  }

  const api = {
    getInstance,
    createSnapshot,
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
            name: 'Create Snapshot ',
            url: null,
          },
        ]}
      />
    ),
  }

  return (
    <CreateSnapshotPage
      instanceId={params.instanceId}
      routes={routes}
      api={api}
      elements={elements}
    />
  )
}

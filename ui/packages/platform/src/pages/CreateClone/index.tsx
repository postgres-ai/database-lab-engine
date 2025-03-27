import { useParams } from 'react-router-dom'

import { CreateClone as CreateClonePage } from '@postgres.ai/shared/pages/CreateClone'

import { ROUTES } from 'config/routes'
import { getInstance } from 'api/instances/getInstance'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { createClone } from 'api/clones/createClone'
import { getClone } from 'api/clones/getClone'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

type Params = {
  org: string
  project?: string
  instanceId: string
}

export const CreateClone = () => {
  const params = useParams<Params>()

  const routes = {
    clone: (cloneId: string) =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.CLONES.CLONE.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
            cloneId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.CLONES.CLONE.createPath({
            ...params,
            cloneId,
          }),
  }

  const api = {
    getSnapshots,
    getInstance,
    createClone,
    getClone,
  }

  const elements = {
    breadcrumbs: (
      <ConsoleBreadcrumbsWrapper
        hasDivider
        org={params.org}
        project={params.project}
        breadcrumbs={[
          { name: 'Database Lab Instances', url: 'instances' },
          { name: 'Instance #' + params.instanceId, url: params.instanceId },
          {
            name: 'Clones',
            url: 'clones',
          },
          { name: 'Create clone', url: null },
        ]}
      />
    ),
  }

  return (
    <CreateClonePage
      instanceId={params.instanceId}
      routes={routes}
      // navRoutes={[ROUTES, ROUTES.INSTANCE, ROUTES.INSTANCE.CLONES.CREATE]}
      api={api}
      elements={elements}
    />
  )
}

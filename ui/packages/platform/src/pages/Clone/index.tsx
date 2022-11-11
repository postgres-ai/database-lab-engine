import { useParams } from 'react-router-dom'

import { Clone as ClonePage } from '@postgres.ai/shared/pages/Clone'

import { getSnapshots } from 'api/snapshots/getSnapshots'
import { getInstance } from 'api/instances/getInstance'
import { getClone } from 'api/clones/getClone'
import { resetClone } from 'api/clones/resetClone'
import { destroyClone } from 'api/clones/destroyClone'
import { updateClone } from 'api/clones/updateClone'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import { ROUTES } from 'config/routes'

type Params = {
  org: string
  project?: string
  instanceId: string
  cloneId: string
}

export const Clone = () => {
  const params = useParams<Params>()

  const routes = {
    instance: () =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.createPath(params),
  }

  const api = {
    getSnapshots,
    getInstance,
    getClone,
    resetClone,
    destroyClone,
    updateClone,
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
            name: 'Clone ' + params.cloneId,
            url: null,
          },
        ]}
      />
    ),
  }

  return (
    <ClonePage
      instanceId={params.instanceId}
      cloneId={params.cloneId}
      routes={routes}
      api={api}
      elements={elements}
    />
  )
}

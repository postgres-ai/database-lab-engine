import { useState } from 'react'
import { useParams } from 'react-router-dom'

import { Instance as InstancePage } from '@postgres.ai/shared/pages/Instance'

import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { ROUTES } from 'config/routes'
import { getInstance } from 'api/instances/getInstance'
import { refreshInstance } from 'api/instances/refreshInstance'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { destroyClone } from 'api/clones/destroyClone'
import { resetClone } from 'api/clones/resetClone'
import { bannersStore } from 'stores/banners'
import { getWSToken } from 'api/instances/getWSToken'
import { getConfig } from 'api/configs/getConfig'
import { getFullConfig } from 'api/configs/getFullConfig'
import { getSeImages } from 'api/configs/getSeImages'
import { testDbSource } from 'api/configs/testDbSource'
import { updateConfig } from 'api/configs/updateConfig'
import { getEngine } from 'api/engine/getEngine'
import { initWS } from 'api/engine/initWS'

type Params = {
  org: string
  project?: string
  instanceId: string
}

export const Instance = () => {
  const params = useParams<Params>()
  const [projectAlias, setProjectAlias] = useState<string>('')

  const routes = {
    createClone: () =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.CLONES.ADD.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.CLONES.ADD.createPath(params),

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
    getInstance,
    getSnapshots,
    destroyClone,
    refreshInstance,
    resetClone,
    getWSToken,
    getConfig,
    getFullConfig,
    getSeImages,
    updateConfig,
    testDbSource,
    getEngine,
    initWS,
  }

  const callbacks = {
    showDeprecatedApiBanner: bannersStore.showDeprecatedApi,
    hideDeprecatedApiBanner: bannersStore.hideDeprecatedApi,
  }

  const instanceTitle = `#${params.instanceId} ${
    projectAlias
      ? `(${projectAlias})`
      : params.project
      ? `(${params.project})`
      : ''
  }`

  const elements = {
    breadcrumbs: (
      <ConsoleBreadcrumbsWrapper
        hasDivider
        org={params.org}
        project={params.project}
        breadcrumbs={[
          { name: 'Database Lab Instances', url: 'instances' },
          {
            name: `Instance ${instanceTitle}`,
            url: null,
          },
        ]}
      />
    ),
  }

  return (
    <InstancePage
      isPlatform
      setProjectAlias={setProjectAlias}
      title={`Database Lab instance ${instanceTitle}`}
      instanceId={params.instanceId}
      routes={routes}
      api={api}
      callbacks={callbacks}
      elements={elements}
    />
  )
}

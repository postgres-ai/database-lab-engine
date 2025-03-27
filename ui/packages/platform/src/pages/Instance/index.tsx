import { useState } from 'react'
import { useParams } from 'react-router-dom'

import { Instance as InstancePage } from '@postgres.ai/shared/pages/Instance'

import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { ROUTES } from 'config/routes'
import { getInstance } from 'api/instances/getInstance'
import { getInstanceRetrieval } from 'api/instances/getInstanceRetrieval'
import { refreshInstance } from 'api/instances/refreshInstance'
import { getSnapshots } from 'api/snapshots/getSnapshots'
import { createSnapshot } from 'api/snapshots/createSnapshot'
import { getBranchSnapshot } from 'api/snapshots/getBranchSnapshot'
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
import { createBranch } from 'api/branches/createBranch'
import { getBranches } from 'api/branches/getBranches'
import { getSnapshotList } from 'api/branches/getSnapshotList'
import { deleteBranch } from 'api/branches/deleteBranch'
import { initWS } from 'api/engine/initWS'
import { destroySnapshot } from 'api/snapshots/destroySnapshot'

type Params = {
  org: string
  project?: string
  instanceId: string
}

export const Instance = ({
  renderCurrentTab,
}: {
  renderCurrentTab?: number
}) => {
  const params = useParams<Params>()
  const [projectAlias, setProjectAlias] = useState<string>('')

  const routes = {
    createBranch: () =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.BRANCHES.ADD.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.BRANCHES.ADD.createPath(params),
    createSnapshot: () =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.SNAPSHOTS.ADD.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.SNAPSHOTS.ADD.createPath(params),
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
    branches: () =>
      params.project
        ? ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.BRANCHES.createPath({
            org: params.org,
            project: params.project,
            instanceId: params.instanceId,
          })
        : ROUTES.ORG.INSTANCES.INSTANCE.BRANCHES.createPath(params),
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
    getInstanceRetrieval,
    getBranchSnapshot,
    getSnapshots,
    createSnapshot,
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
    createBranch,
    getBranches,
    getSnapshotList,
    deleteBranch,
    destroySnapshot,
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
      renderCurrentTab={renderCurrentTab}
    />
  )
}

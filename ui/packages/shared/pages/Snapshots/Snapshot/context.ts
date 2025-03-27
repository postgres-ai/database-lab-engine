import { createStrictContext } from '@postgres.ai/shared/utils/react'

import { Api } from './stores/Main'
import { Stores } from './useCreatedStores'

export type Host = {
  instanceId: string
  snapshotId: string
  routes: {
    snapshot: () => string
    branch: (branchName: string) => string
    clone: (cloneId: string) => string
  }
  api: Api
  elements: {
    breadcrumbs: React.ReactNode
  }
}

export const { useStrictContext: useHost, Provider: HostProvider } =
  createStrictContext<Host>()

export const { useStrictContext: useStores, Provider: StoresProvider } =
  createStrictContext<Stores>()

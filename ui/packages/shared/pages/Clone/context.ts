import { createStrictContext } from '@postgres.ai/shared/utils/react'

import { Api } from './stores/Main'
import { Stores } from './useCreatedStores'

export type Host = {
  instanceId: string
  cloneId: string
  routes: {
    instance: () => string
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

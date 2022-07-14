/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { createStrictContext } from '@postgres.ai/shared/utils/react'

import { Api } from './stores/Main'
import { Stores } from './useCreatedStores'

export type Host = {
  instanceId: string
  routes: {
    createClone: () => string
    clone: (cloneId: string) => string
  }
  api: Api
  title: string
  callbacks?: {
    showDeprecatedApiBanner: () => void
    hideDeprecatedApiBanner: () => void
  }
  elements: {
    breadcrumbs: React.ReactNode
  },
  wsHost?: string
}

// Host context.
export const { useStrictContext: useHost, Provider: HostProvider } =
  createStrictContext<Host>()

// Stores context.
export const { useStrictContext: useStores, Provider: StoresProvider } =
  createStrictContext<Stores>()

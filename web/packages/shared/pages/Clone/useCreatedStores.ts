import { useMemo } from 'react'

import { MainStore } from './stores/Main'

import { Host } from './context'

export const useCreatedStores = (host: Host) => ({
  main: useMemo(() => new MainStore(host.api), []),
})

export type Stores = ReturnType<typeof useCreatedStores>

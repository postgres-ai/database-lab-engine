import { useMemo } from 'react'

import { MainStore } from './stores/Main'
import { Host } from './context'

export const useCreatedStores = (api: Host["api"]) => ({
  main: useMemo(() => new MainStore(api), []),
})

export type Stores = ReturnType<typeof useCreatedStores>

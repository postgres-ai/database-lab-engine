import { useMemo } from 'react'

import { MainStore, MainStoreApi } from './stores/Main'

export const useCreatedStores = (api: MainStoreApi) => ({
  main: useMemo(() => new MainStore(api), []),
})

export type Stores = ReturnType<typeof useCreatedStores>

export type { MainStoreApi }

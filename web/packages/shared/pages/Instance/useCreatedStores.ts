import { useMemo } from 'react'

import { MainStore } from './stores/Main'
import { ClonesModalStore } from './stores/ClonesModal'
import { SnapshotsModalStore } from './stores/SnapshotsModal'
import { Host } from './context'

export const useCreatedStores = (host: Host) => ({
  main: useMemo(() => new MainStore(host.api), []),
  clonesModal: useMemo(() => new ClonesModalStore(), []),
  snapshotsModal: useMemo(() => new SnapshotsModalStore(), []),
})

export type Stores = ReturnType<typeof useCreatedStores>

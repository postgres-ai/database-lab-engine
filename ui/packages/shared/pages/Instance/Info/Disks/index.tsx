/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { observer } from 'mobx-react-lite'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'

import { Section } from '../components/Section'

import { PoolSection } from './PoolSection'
import { DatasetInfo } from './PoolSection/DatasetRow'

export const Disks = observer(() => {
  const stores = useStores()

  const { instance, snapshots } = stores.main
  if (!instance) return null
  if (!snapshots) return null

  const pools = instance.state?.pools

  if (pools && pools.length > 0) {
    const poolGroups: Record<string, typeof pools[number][]> = {}
    for (const pool of pools) {
      const slashIdx = pool.name.indexOf('/')
      const poolName =
        slashIdx !== -1 ? pool.name.slice(0, slashIdx) : pool.name
      if (!poolGroups[poolName]) poolGroups[poolName] = []
      poolGroups[poolName].push(pool)
    }

    return (
      <Section title="Disks">
        {Object.entries(poolGroups).map(([poolName, poolList]) => {
          const firstPool = poolList[0]

          const datasets: DatasetInfo[] = poolList.map((pool) => {
            const slashIdx = pool.name.indexOf('/')
            const datasetName =
              slashIdx !== -1 ? pool.name.slice(slashIdx + 1) : pool.name

            const datasetSnapshots =
              snapshots.data?.filter((s) => s.pool === pool.name) ?? []

            return {
              id: pool.name,
              name: datasetName,
              showName: slashIdx !== -1,
              status: pool.status,
              mode: pool.mode,
              usedDataSize: pool.fileSystem.used,
              clonesCount: pool.cloneList.length,
              snapshotsCount: datasetSnapshots.length,
              refreshingStartDate:
                instance.state?.retrieving?.lastRefresh ?? null,
            }
          })

          return (
            <PoolSection
              key={poolName}
              poolName={poolName}
              totalSize={firstPool.fileSystem.size}
              freeSize={firstPool.fileSystem.free}
              datasets={datasets}
            />
          )
        })}
      </Section>
    )
  }

  if (instance.state?.fileSystem) {
    const allSnapshots = snapshots.data ?? []

    const dataset: DatasetInfo = {
      id: null,
      name: 'Main',
      showName: false,
      status: 'active',
      mode: 'zfs',
      usedDataSize: instance.state.fileSystem.used,
      clonesCount:
        instance.state?.clones?.length ??
        instance.state.cloning.clones.length,
      snapshotsCount: allSnapshots.length,
      refreshingStartDate:
        instance.state?.retrieving?.lastRefresh ?? null,
    }

    return (
      <Section title="Disks">
        <PoolSection
          poolName="Main"
          totalSize={instance.state.fileSystem.size}
          freeSize={instance.state.fileSystem.free}
          datasets={[dataset]}
        />
      </Section>
    )
  }

  return (
    <Section title="Disks">
      <>
        Disk information is <strong>unavailable</strong>.
      </>
    </Section>
  )
})

/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { observer } from 'mobx-react-lite'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'

import { Section } from '../components/Section'

import { Disk } from './Disk'

export const Disks = observer(() => {
  const stores = useStores()

  const { instance, snapshots } = stores.main
  if (!instance) return null
  if (!snapshots) return null

  return (
    <Section title="Disks">
      {instance.state.pools?.map((pool) => {
        return (
          <Disk
            key={pool.name}
            status={pool.status}
            name={pool.name}
            id={pool.name}
            mode={pool.mode}
            clonesCount={pool.cloneList.length}
            snapshotsCount={
              snapshots.data?.filter((snapshot) => snapshot.pool === pool.name)
                .length ?? 0
            }
            totalDataSize={pool.fileSystem.size}
            usedDataSize={pool.fileSystem.used}
            freeDataSize={pool.fileSystem.free}
            refreshingStartDate={instance.state.retrieving?.lastRefresh ?? null}
          />
        )
      }) ??
        (instance.state.fileSystem && (
          <Disk
            status={'active'}
            name={'Main'}
            id={null}
            mode="ZFS"
            clonesCount={instance.state?.clones?.length ?? instance.state.cloning.clones.length}
            snapshotsCount={snapshots.data?.length ?? 0}
            totalDataSize={instance.state.fileSystem.size}
            usedDataSize={instance.state.fileSystem.used}
            freeDataSize={instance.state.fileSystem.free}
            refreshingStartDate={null}
          />
        )) ?? (
          <>
            Disk information is <strong>unavailable</strong>.
          </>
        )}
    </Section>
  )
})

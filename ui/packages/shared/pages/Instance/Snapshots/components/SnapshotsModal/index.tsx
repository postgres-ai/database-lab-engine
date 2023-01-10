/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { observer } from 'mobx-react-lite'
import { makeStyles } from '@material-ui/core'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { Modal as ModalBase } from '@postgres.ai/shared/components/Modal'
import { isSameDayUTC } from '@postgres.ai/shared/utils/date'

import { Tags } from '@postgres.ai/shared/pages/Instance/components/Tags'
import { ModalReloadButton } from '@postgres.ai/shared/pages/Instance/components/ModalReloadButton'
import { SnapshotsTable } from '@postgres.ai/shared/pages/Instance/Snapshots/components/SnapshotsTable'

import { getTags } from './utils'

const useStyles = makeStyles(
  {
    root: {
      fontSize: '14px',
      marginTop: 0,
    },
    emptyStub: {
      marginTop: '16px',
    },
  },
  { index: 1 },
)

export const SnapshotsModal = observer(() => {
  const classes = useStyles()
  const stores = useStores()

  const { snapshots } = stores.main
  if (!snapshots.data) return null

  const filteredSnapshots = snapshots.data.filter((snapshot) => {
    const isMatchedByDate =
      !stores.snapshotsModal.date ||
      isSameDayUTC(snapshot.dataStateAtDate, stores.snapshotsModal.date)

    const isMatchedByPool =
      !stores.snapshotsModal.pool ||
      snapshot.pool === stores.snapshotsModal.pool

    return isMatchedByDate && isMatchedByPool
  })

  const isEmpty = !filteredSnapshots.length

  return (
    <ModalBase
      isOpen={stores.snapshotsModal.isOpenModal}
      onClose={stores.snapshotsModal.closeModal}
      title={`Snapshots (${filteredSnapshots.length})`}
      classes={{ content: classes.root }}
      size="md"
      titleRightContent={
        <ModalReloadButton
          isReloading={stores.main.snapshots.isLoading}
          onReload={stores.main.reloadSnapshots}
        />
      }
      headerContent={
        <Tags
          data={getTags({
            date: stores.snapshotsModal.date,
            pool: stores.snapshotsModal.pool,
          })}
        />
      }
    >
      {!isEmpty ? (
        <SnapshotsTable />
      ) : (
        <p className={classes.emptyStub}>No snapshots found</p>
      )}
    </ModalBase>
  )
})

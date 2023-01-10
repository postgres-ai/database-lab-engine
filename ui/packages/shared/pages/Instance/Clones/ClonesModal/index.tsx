/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/core'
import { observer } from 'mobx-react-lite'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { Modal } from '@postgres.ai/shared/components/Modal'
import { ClonesList } from '@postgres.ai/shared/pages/Instance/Clones/ClonesList'
import { Tags } from '@postgres.ai/shared/pages/Instance/components/Tags'
import { ModalReloadButton } from '@postgres.ai/shared/pages/Instance/components/ModalReloadButton'

import { getTags } from './utils'

const useStyles = makeStyles(
  {
    root: {
      marginTop: 0,
    },
  },
  { index: 1 },
)

export const ClonesModal = observer(() => {
  const classes = useStyles()
  const stores = useStores()

  const { instance } = stores.main
  if (!instance) return null

  const { pool, snapshotId, isOpenModal, closeModal } = stores.clonesModal

  return (
    <Modal
      title={`Clones (${instance.state.cloning.clones.length})`}
      isOpen={isOpenModal}
      onClose={closeModal}
      size="md"
      titleRightContent={
        <ModalReloadButton
          isReloading={stores.main.isReloadingClones}
          onReload={stores.main.reloadClones}
        />
      }
      headerContent={
        <Tags
          data={getTags({
            pool,
            snapshotId,
          })}
        />
      }
      classes={{ content: classes.root }}
    >
      <ClonesList
        isDisabled={false}
        clones={instance.state.cloning.clones.filter((clone) => {
          const isMatchedByPool = !pool || pool === clone.snapshot?.pool
          const isMatchedBySnapshot =
            !snapshotId || snapshotId === clone.snapshot?.id
          return isMatchedByPool && isMatchedBySnapshot
        })}
        emptyStubText="No clones found"
      />
    </Modal>
  )
})

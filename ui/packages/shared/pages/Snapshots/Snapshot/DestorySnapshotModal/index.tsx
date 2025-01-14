/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'
import { makeStyles } from '@material-ui/core'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { ImportantText } from '@postgres.ai/shared/components/ImportantText'
import { Text } from '@postgres.ai/shared/components/Text'
import { destroySnapshot as destroySnapshotAPI } from '@postgres.ai/ce/src/api/snapshots/destroySnapshot'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'
import { useCreatedStores } from '../useCreatedStores'

type Props = {
  snapshotId: string
  isOpen: boolean
  onClose: () => void
  afterSubmitClick: () => void
}

const useStyles = makeStyles(
  {
    errorMessage: {
      color: 'red',
      marginTop: '10px',
    },
  },
  { index: 1 },
)

export const DestroySnapshotModal = ({
  snapshotId,
  isOpen,
  onClose,
  afterSubmitClick,
}: Props) => {
  const classes = useStyles()
  const props = { api: { destroySnapshot: destroySnapshotAPI } }
  const stores = useCreatedStores(props.api)
  const { destroySnapshot } = stores.main
  const [deleteError, setDeleteError] = useState(null)

  const handleClose = () => {
    setDeleteError(null)
    onClose()
  }

  const handleClickDestroy = () => {
    destroySnapshot(snapshotId).then((res) => {
      if (res?.error?.message) {
        setDeleteError(res.error.message)
      } else {
        afterSubmitClick()
        handleClose()
      }
    })
  }

  return (
    <Modal title={'Confirmation'} onClose={handleClose} isOpen={isOpen}>
      <Text>
        Are you sure you want to destroy snapshot{' '}
        <ImportantText>{snapshotId}</ImportantText>? This action cannot be
        undone.
      </Text>
      {deleteError && <p className={classes.errorMessage}>{deleteError}</p>}
      <SimpleModalControls
        items={[
          {
            text: 'Cancel',
            onClick: handleClose,
          },
          {
            text: 'Destroy snapshot',
            variant: 'primary',
            onClick: handleClickDestroy,
          },
        ]}
      />
    </Modal>
  )
}
